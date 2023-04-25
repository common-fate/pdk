package command

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/common-fate/clio"
	"github.com/common-fate/pdk/pkg/cfngen"
	"github.com/common-fate/pdk/pkg/iamp"
	"github.com/common-fate/pdk/pkg/pythonconfig"
	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
	"github.com/pkg/errors"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/urfave/cli/v2"
)

type Provider struct {
	Publisher     string `json:"publisher"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	SchemaVersion string `json:"schema_version"`
}

func (p Provider) String() string {
	return fmt.Sprintf("%s/%s@%s (schema %s)", p.Publisher, p.Name, p.Version, p.SchemaVersion)
}

type localDependency struct {
	// Name of the package e.g. 'commonfate_provider'
	Name string
	// Path on disk for the package e.g. '../commonfate-provider-core/commonfate_provider'
	Path string
}

func parseLocalDependency(input string) (localDependency, error) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return localDependency{}, fmt.Errorf("invalid local dependency %s: must be in packagename=path format", input)
	}
	ld := localDependency{
		Name: parts[0],
		Path: parts[1],
	}
	return ld, nil
}

type PackageFlagOpts struct {
	LocalDependency []string
}

func PackageAndZip(ctx context.Context, providerPath string, flagOpts PackageFlagOpts) error {
	dist := filepath.Join(providerPath, "dist")

	// clean the dist folder
	err := os.RemoveAll(dist)
	if err != nil {
		return err
	}
	err = os.Mkdir(dist, os.ModePerm)
	if err != nil {
		return err
	}

	fpath := filepath.Join(dist, "handler.zip")

	configFile := filepath.Join(providerPath, "provider.toml")
	cfg, err := pythonconfig.LoadFile(configFile)
	if err != nil {
		return err
	}

	cmd := exec.Command(".venv/bin/commonfate-provider-py", "schema")

	var outb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Println(outb.String())
		return err
	}

	provider := Provider{
		Publisher: cfg.Publisher,
		Name:      cfg.Name,
		Version:   cfg.Version,
		// in future, this will be determined by reading
		// the schema from the registry and determining whether
		// it has changed
		SchemaVersion: "v1",
	}

	var schema map[string]any

	err = json.Unmarshal(outb.Bytes(), &schema)
	if err != nil {
		return err
	}

	// add the $id field to the schema in the format
	// https://registry.commonfate.io/schema/{publisher}/{name}/{schema_version}
	//
	// Note that schema version is currently fixed at "v1" always - in future
	// we'll have a way to update the schema version based on detecting whether
	// it's different to the latest version in our registry.
	schemaID := fmt.Sprintf("https://registry.commonfate.io/schema/%s/%s/%s", provider.Publisher, provider.Name, provider.SchemaVersion)
	schema["$id"] = schemaID

	schemaMarshalled, err := json.Marshal(schema)
	if err != nil {
		return err
	}

	shemaFile := "dist/schema.json"
	err = os.WriteFile(shemaFile, schemaMarshalled, 0644)
	if err != nil {
		return err
	}

	clio.Successf("exported Provider schema %s to %s", schemaID, shemaFile)

	var localDependencies []localDependency

	for _, localDepInput := range flagOpts.LocalDependency {
		// parse the local dependency input - it's in the format
		// package_name=../path/to/package
		ld, err := parseLocalDependency(localDepInput)
		if err != nil {
			return err
		}
		localDependencies = append(localDependencies, ld)
	}

	err = PackageProvider(PackageProviderOpts{
		ProviderPath:      providerPath,
		Provider:          provider,
		OutputPath:        fpath,
		LocalDependencies: localDependencies,
	})
	if err != nil {
		return err
	}

	clio.Successf("zipped provider")

	// generate CloudFormation templates for any roles in the `roles` directory
	err = generateAccessRoleTemplates(providerPath, cfg)
	if err != nil {
		return err
	}

	// unmarshalling again here is a little messy
	var providerSchema providerregistrysdk.Schema
	err = json.Unmarshal(outb.Bytes(), &providerSchema)
	if err != nil {
		return err
	}

	// create the CloudFormation template for the Provider
	cloudformationTemplate, err := cfngen.Generate(cfg, providerSchema)
	if err != nil {
		return err
	}

	cfnPath := filepath.Join(dist, "cloudformation.json")

	err = os.WriteFile(cfnPath, cloudformationTemplate, 0644)
	if err != nil {
		return err
	}
	clio.Successf("generated cloudformation template: %s", cfnPath)

	clio.Successf("packaged %s to %s", provider, fpath)

	return nil
}

var Package = cli.Command{
	Name: "package",
	Flags: []cli.Flag{
		&cli.PathFlag{Name: "path", Value: ".", Usage: "The path to the folder containing your provider code e.g ./cf-provider-example"},
		&cli.StringSliceFlag{Name: "local-dependency", Usage: "(For development use) Add a local python package to the zip archive, e.g. commonfate_provider=../commonfate-provider-core/commonfate_provider"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		providerPath := c.Path("path")

		localDependency := c.StringSlice("local-dependency")

		err := PackageAndZip(ctx, providerPath, PackageFlagOpts{
			LocalDependency: localDependency,
		})
		if err != nil {
			return err
		}

		return nil
	},
}

func generateAccessRoleTemplates(dir string, pconfig pythonconfig.Config) error {
	roleDir := path.Join(dir, "roles")
	_, err := os.Stat(roleDir)
	if os.IsNotExist(err) {
		// provider does not have any access roles
		return nil
	}

	files, err := os.ReadDir(roleDir)
	if err != nil {
		return err
	}

	outdir := path.Join(dir, "dist", "roles")
	err = os.MkdirAll(outdir, 0755)
	if err != nil {
		return err
	}

	for _, f := range files {
		roleData, err := os.ReadFile(path.Join(roleDir, f.Name()))
		if err != nil {
			return err
		}

		var policy iamp.Policy

		err = json.Unmarshal(roleData, &policy)
		if err != nil {
			return err
		}

		name := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))

		template, err := cfngen.GenerateAccessRole(pconfig, name, policy)
		if err != nil {
			return err
		}
		outputPath := path.Join(outdir, f.Name())
		err = os.WriteFile(outputPath, template, 0644)
		if err != nil {
			return err
		}

		clio.Infof("generated Access Role CloudFormation template: %s", outputPath)
	}
	return nil
}

type PackageProviderOpts struct {
	ProviderPath      string
	OutputPath        string
	Provider          Provider
	LocalDependencies []localDependency
}

// PackageProvider creates a zip archive bundle for the provider.
func PackageProvider(opts PackageProviderOpts) error {
	if _, err := os.Stat(opts.OutputPath); !errors.Is(err, os.ErrNotExist) {
		clio.Infof("deleting existing zip %s", opts.OutputPath)
		err := os.Remove(opts.OutputPath)
		if err != nil {
			return errors.Wrapf(err, "removing existing zip %s", opts.OutputPath)
		}
	}

	pythonDepFolder := "pythondeps"
	// remove the Python dependency folder if it exists
	if _, err := os.Stat(pythonDepFolder); !errors.Is(err, os.ErrNotExist) {
		clio.Infof("deleting python dependency folder %s", pythonDepFolder)
		err := os.RemoveAll(pythonDepFolder)
		if err != nil {
			return errors.Wrapf(err, "removing python dependency folder %s", pythonDepFolder)
		}
	}

	// package the provider-specific Python dependencies into the zip.
	clio.Info("packaging Provider Python dependencies")
	fullReqFile := path.Join(opts.ProviderPath, "requirements.txt")
	cmd := exec.Command(".venv/bin/pip", "install", "--platform", "manylinux2014_x86_64", "--only-binary", ":all:", "-r", fullReqFile, "--target", pythonDepFolder)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	clio.Infof("creating destination path %s", opts.OutputPath)

	destinationFile, err := os.Create(opts.OutputPath)
	if err != nil {
		return err
	}
	myZip := zip.NewWriter(destinationFile)

	clio.Infof("zipping %s", pythonDepFolder)

	var localPackageNames []string

	for _, ld := range opts.LocalDependencies {
		localPackageNames = append(localPackageNames, ld.Name)
	}

	err = addToZip(AddToZipOpts{
		Writer:     myZip,
		PathToZip:  pythonDepFolder,
		TrimPrefix: "pythondeps/",
		Ignore:     ignore.CompileIgnoreLines(localPackageNames...),
	})
	if err != nil {
		return err
	}

	gitignore, err := ignore.CompileIgnoreFileAndLines(filepath.Join(opts.ProviderPath, ".gitignore"), "dist", "pythondeps")
	if err != nil {
		return err
	}

	clio.Infof("zipping %s", opts.ProviderPath)

	// add the provider itself. This is the module written by the
	// Provider developer.
	err = addToZip(AddToZipOpts{
		Writer:              myZip,
		PathToZip:           opts.ProviderPath,
		OnlyTheseExtensions: []string{".py"},
		Ignore:              gitignore,
		ZippedPathPrefix:    "commonfate_provider_dist",
	})
	if err != nil {
		return err
	}

	// add the manifest.json file with the metadata about the provider.
	w, err := myZip.Create("commonfate_provider_dist/manifest.json")
	if err != nil {
		return err
	}

	err = json.NewEncoder(w).Encode(opts.Provider)
	if err != nil {
		return err
	}

	// add any local Python packages to the provider archive
	// these allow for development versions of our commonfate_provider
	// library to be included, rather than using an official release.
	for _, ldp := range opts.LocalDependencies {
		// the ldp variable looks like this
		// '../../some/path/to/package'
		//
		// turn it into an absolute path -
		// /Users/myuser/code/some/path/to/package
		abs, err := filepath.Abs(ldp.Path)
		clio.Info(abs)
		if err != nil {
			return err
		}

		clio.Infof("adding local Python dependency to zip: %s", abs)

		// create a directory for the local package
		_, err = myZip.Create(ldp.Name + "/")
		if err != nil {
			return err
		}

		err = addToZip(AddToZipOpts{
			Writer:     myZip,
			PathToZip:  abs,
			TrimPrefix: "/",
		})
		if err != nil {
			return err
		}
	}

	err = myZip.Close()
	if err != nil {
		return err
	}
	return nil
}

type AddToZipOpts struct {
	Writer    *zip.Writer
	PathToZip string
	Ignore    *ignore.GitIgnore

	OnlyTheseExtensions []string

	// ZippedPathPrefix sets a prefix to the path in the zip file if specified.
	ZippedPathPrefix string

	// TrimPrefix trims the file prefix,
	// e.g. pythondeps/packagename -> packagename
	TrimPrefix string
}

func addToZip(opts AddToZipOpts) error {
	return filepath.Walk(opts.PathToZip, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if opts.Ignore != nil {
			if matches, pattern := opts.Ignore.MatchesPathHow(filePath); matches {
				clio.Debugf("skipping %s (matched ignore pattern %s)", filePath, pattern.Pattern)
				return nil
			}
		}

		if len(opts.OnlyTheseExtensions) > 0 {
			var matchedExt bool
			for _, ext := range opts.OnlyTheseExtensions {
				if filepath.Ext(filePath) == ext {
					matchedExt = true
					break
				}
			}
			if !matchedExt {
				clio.Debugf("skipping %s (didn't match file extensions %s)", filePath, strings.Join(opts.OnlyTheseExtensions, ", "))
				return nil
			}
		}

		relPath := strings.TrimPrefix(filePath, filepath.Dir(opts.PathToZip))
		clio.Debug("pre trim prefix = ", relPath)
		relPath = strings.TrimPrefix(relPath, opts.TrimPrefix)
		clio.Debug("post trim prefix = ", relPath)
		if opts.ZippedPathPrefix != "" {
			relPath = filepath.Join(opts.ZippedPathPrefix, relPath)
		}

		clio.Debugf("zipping %s to %s", filePath, relPath)

		zipFile, err := opts.Writer.Create(relPath)
		if err != nil {
			return err
		}
		fsFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		_, err = io.Copy(zipFile, fsFile)
		if err != nil {
			return err
		}
		return nil
	})
}
