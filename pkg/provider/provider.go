package provider

import (
	"path"
	"time"

	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
)

type S3Paths struct {
	Publisher string
	Name      string
	Version   string
}

func (p S3Paths) Handler() string {
	return path.Join(p.Publisher, p.Name, p.Version, "handler.zip")
}
func (p S3Paths) CloudformationTemplate() string {
	return path.Join(p.Publisher, p.Name, p.Version, "cloudformation.json")
}
func (p S3Paths) Readme() string {
	return path.Join(p.Publisher, p.Name, p.Version, "readme.md")
}

func (p S3Paths) RoleCloudformationTemplate(name string) string {
	return path.Join(p.Publisher, p.Name, p.Version, "roles", name)
}

// The publishing type represents a provider which is in the process of being published
// when publishing is complete, and validated, a Provider type will be created from the outcome of publishing.
type Publishing struct {
	Publisher string                               `json:"publisher" dynamodbav:"publisher"`
	Name      string                               `json:"name" dynamodbav:"name"`       // name should be the id, limited characters, no space
	Version   string                               `json:"version" dynamodbav:"version"` // specified by the users (semver)
	IsDev     bool                                 `json:"isDev" dynamodbav:"isDev"`     // isDev flag is used to differentiate and filter dev providers
	Schema    providerregistrysdk.Schema           `json:"schema" dynamodbav:"schema"`
	Meta      providerregistrysdk.ProviderMetaInfo `json:"meta" dynamodbav:"meta"`
	// The file paths that were requested for upload
	RoleFiles []string `json:"roleFiles" dynamodbav:"roleFiles"`
}

func (p Publishing) S3Paths() S3Paths {
	return S3Paths{Publisher: p.Publisher, Name: p.Name, Version: p.Version}
}

type Provider struct {
	Publisher     string                               `json:"publisher" dynamodbav:"publisher"`
	Name          string                               `json:"name" dynamodbav:"name"`       // name should be the id, limited characters, no space
	IsDev         bool                                 `json:"isDev" dynamodbav:"isDev"`     // isDev flag is used to differentiate and filter dev providers
	Version       string                               `json:"version" dynamodbav:"version"` // specified by the users (semver)
	Meta          providerregistrysdk.ProviderMetaInfo `json:"meta" dynamodbav:"meta"`
	Latest        bool                                 `json:"latest" dynamodbav:"latest"`
	Schema        providerregistrysdk.Schema           `json:"schema" dynamodbav:"schema"`
	CreatedBy     string                               `json:"createdBy" dynamodbav:"createdBy"`
	LastUpdatedBy string                               `json:"lastUpdatedBy" dynamodbav:"lastUpdatedBy"`
	CreatedAt     time.Time                            `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt     time.Time                            `json:"updatedAt" dynamodbav:"updatedAt"`
}

func (p Provider) S3Paths() S3Paths {
	return S3Paths{Publisher: p.Publisher, Name: p.Name, Version: p.Version}
}

func (p Provider) ToAPI(assetsBucketName string) providerregistrysdk.ProviderDetail {
	return providerregistrysdk.ProviderDetail{
		Name:             p.Name,
		Publisher:        p.Publisher,
		Version:          p.Version,
		Schema:           p.Schema,
		Meta:             &p.Meta,
		LambdaAssetS3Arn: path.Join(assetsBucketName, p.S3Paths().Handler()),
		CfnTemplateS3Arn: path.Join(assetsBucketName, p.S3Paths().CloudformationTemplate()),
		CreatedAt:        p.CreatedAt.String(),
		UpdatedAt:        p.UpdatedAt.String(),
	}
}
