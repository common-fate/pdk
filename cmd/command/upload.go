package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/common-fate/clio"
	"github.com/common-fate/pdk/pkg/client"
	"github.com/common-fate/pdk/pkg/pythonconfig"
	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

type Paths struct {
	ProviderPath string
}

func (p Paths) Handler() string {
	return path.Join(p.ProviderPath, "dist", "handler.zip")
}
func (p Paths) CloudformationTemplate() string {
	return path.Join(p.ProviderPath, "dist", "cloudformation.json")
}
func (p Paths) Readme() string {
	return path.Join(p.ProviderPath, "README.md")
}
func (p Paths) Schema() string {
	return path.Join(p.ProviderPath, "dist", "schema.json")
}
func (p Paths) RoleTemplateFolder() string {
	return path.Join(p.ProviderPath, "roles")
}
func (p Paths) RoleTemplate(role string) string {
	return path.Join(p.ProviderPath, "roles", role)
}

// CheckFilesExist is used to assert that the required files exist prior to starting a publish workflow
func CheckFilesExist(providerPath string) error {
	p := Paths{ProviderPath: providerPath}

	_, err := os.Stat(p.Handler())
	if os.IsNotExist(err) {
		return fmt.Errorf("expected to find handler zip at the following path: %s", p.Handler())
	}

	_, err = os.Stat(p.CloudformationTemplate())
	if os.IsNotExist(err) {
		return fmt.Errorf("expected to find cloudformation template at the following path: %s", p.Readme())
	}

	_, err = os.Stat(p.Readme())
	if os.IsNotExist(err) {
		return fmt.Errorf("expected to find readme at the following path: %s", p.Readme())
	}
	_, err = os.Stat(p.Schema())
	if os.IsNotExist(err) {
		return fmt.Errorf("expected to find schema at the following path: %s", p.Schema())
	}
	return nil
}

// DetectRoleTemplateFiles will look for files in the roles folder which have the .json extension
func DetectRoleTemplateFiles(roleTemplateFolder string) ([]string, error) {
	_, err := os.Stat(roleTemplateFolder)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	roleTemplatesFiles := []string{}
	err = filepath.Walk(roleTemplateFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			roleTemplatesFiles = append(roleTemplatesFiles, info.Name())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return roleTemplatesFiles, nil
}
func contains(arr []string, target string) bool {
	for _, element := range arr {
		if element == target {
			return true
		}
	}
	return false
}
func VerifyUserBelongsToPublisher(ctx context.Context, publisher string) error {
	// get me
	registryclient, err := client.NewWithAuthToken(ctx)
	if err != nil {
		return err
	}
	meResponse, err := registryclient.UserGetMeWithResponse(ctx)
	if err != nil {
		return err
	}

	// check list of publishers contains the publisher
	if !contains(meResponse.JSON200.Publishers, publisher) {
		return fmt.Errorf("you are not a member of this publisher: '%s'. You can try to create it now by running 'pdk publisher create'", publisher)
	}

	return nil
}

type UploadFlagOpts struct {
	Dev bool
}

func UploadProvider(ctx context.Context, providerPath string, flagOpts UploadFlagOpts) error {
	err := CheckFilesExist(providerPath)
	if err != nil {
		return err
	}
	paths := Paths{
		ProviderPath: providerPath,
	}
	schemaBytes, err := os.ReadFile(paths.Schema())
	if err != nil {
		return err
	}

	var schema providerregistrysdk.Schema
	err = json.Unmarshal(schemaBytes, &schema)
	if err != nil {
		return err
	}

	registryclient, err := client.NewWithAuthToken(ctx)
	if err != nil {
		return err
	}

	configFile := filepath.Join(providerPath, "provider.toml")
	pconfig, err := pythonconfig.LoadFile(configFile)
	if err != nil {
		return err
	}

	err = VerifyUserBelongsToPublisher(ctx, pconfig.Publisher)
	if err != nil {
		return err
	}

	roleTemplateFileNames, err := DetectRoleTemplateFiles(paths.RoleTemplateFolder())
	if err != nil {
		return err
	}

	isDev := flagOpts.Dev
	publishRequest := providerregistrysdk.UserPublishProviderJSONRequestBody{
		Dev:       &isDev,
		Name:      pconfig.Name,
		Publisher: pconfig.Publisher,
		Version:   pconfig.Version,
		Meta:      pconfig.Meta.ToAPI(),
		Schema:    schema,
		RoleFiles: roleTemplateFileNames,
	}

	res, err := registryclient.UserPublishProviderWithResponse(ctx, publishRequest)
	if err != nil {
		return err
	}

	var g errgroup.Group
	g.Go(func() error {
		return uploadToS3(res.JSON200.CloudformationTemplateUploadUrl, paths.CloudformationTemplate())
	})
	g.Go(func() error {
		return uploadToS3(res.JSON200.LambdaHandlerUploadUrl, paths.Handler())
	})
	g.Go(func() error {
		return uploadToS3(res.JSON200.ReadmeUploadUrl, paths.Readme())
	})
	for role, url := range res.JSON200.RoleTemplateUploadURLs {
		r := role
		u := url
		g.Go(func() error {
			return uploadToS3(u, paths.RoleTemplate(r))
		})
	}

	err = g.Wait()
	if err != nil {
		return err
	}
	_, err = registryclient.UserCompletePublishProviderWithResponse(ctx, providerregistrysdk.Provider{
		Name:      pconfig.Name,
		Publisher: pconfig.Publisher,
		Version:   pconfig.Version,
	})
	if err != nil {
		return err
	}

	clio.Success("Successfully published provider")

	return nil
}

var UploadCommand = cli.Command{
	Name:  "upload",
	Usage: "Upload a provider to the registry",
	Flags: []cli.Flag{
		&cli.PathFlag{Name: "path", Value: ".", Usage: "The path to the folder containing your provider code e.g ./cf-provider-example"},
		&cli.BoolFlag{Hidden: true, Name: "dev", Usage: "Pass this flag to hide provider from production registry"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		providerPath := c.Path("path")
		err := UploadProvider(ctx, providerPath, UploadFlagOpts{
			Dev: c.Bool("dev"),
		})
		if err != nil {
			return err
		}
		return nil
	},
}

func uploadToS3(url string, name string) error {
	clio.Infof("Uploading file: %s", name)

	f, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(f))
	if err != nil {
		return err
	}

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	// TODO we should be able to skip a file if it already exists in s3, based on the error response.
	// this allows for repeated tries to publish the same provider version, in the case something goes wrong.
	// But to keep it simple for now, it's not implemented
	if resp.StatusCode != http.StatusOK {
		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("failed to upload file to S3 with error: %s", body)
	}
	clio.Successf("Successfully uploaded file: %s", name)
	return nil
}
