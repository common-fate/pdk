package command

import (
	"github.com/common-fate/clio"
	"github.com/urfave/cli/v2"
)

var PublishCommand = cli.Command{
	Name:  "publish",
	Usage: "Publish will package and upload the provider in the provided path argument",
	Flags: []cli.Flag{
		&cli.PathFlag{Name: "path", Value: ".", Usage: "The path to the folder containing your provider code e.g ./cf-provider-example"},
		&cli.BoolFlag{Hidden: true, Name: "dev", Usage: "Pass this flag to hide provider from production registry"},
		&cli.StringSliceFlag{Name: "local-dependency", Usage: "(For development use) Add a local python package to the zip archive, e.g. commonfate_provider=../commonfate-provider-core/commonfate_provider"},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		providerPath := c.Path("path")

		clio.Debugf("packaging a provider in path %s", providerPath)

		err := PackageAndZip(ctx, providerPath, PackageFlagOpts{
			LocalDependency: c.StringSlice("local-dependency"),
		})
		if err != nil {
			return err
		}

		err = UploadProvider(ctx, providerPath, UploadFlagOpts{

			Dev: c.Bool("dev"),
		})
		if err != nil {
			return err
		}

		return nil
	},
}
