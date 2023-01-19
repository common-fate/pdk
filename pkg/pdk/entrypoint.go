package pdk

import (
	"os"

	"github.com/common-fate/clio"
	"github.com/common-fate/pdk/internal/build"
	"github.com/urfave/cli/v2"
)

// Prevent issues where these flags are initialised in some part of the program then used by another part
// For our use case, we need fresh copies of these flags in the app and in the assume command
// we use this to allow flags to be set on either side of the profile arg e.g `assume -c profile-name -r ap-southeast-2`
func GlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "role-name", Usage: "Use this in conjunction with --sso, the role-name"},
	}
}

func GetCliApp() *cli.App {
	cli.VersionPrinter = func(c *cli.Context) {
		clio.Log("print version")
		// clio.Log(banners.WithVersion(banners.Assume()))
	}

	app := &cli.App{
		Name:        "pdk",
		Writer:      os.Stderr,
		Usage:       "https://granted.dev",
		UsageText:   "pdk [options][command]",
		Version:     build.Version,
		HideVersion: false,
		Flags:       GlobalFlags(),
		Action:      PkdCommand,
		Before: func(c *cli.Context) error {

			clio.SetLevelFromEnv("GRANTED_LOG")
			if c.Bool("verbose") {
				clio.SetLevelFromString("debug")
			}

			return nil
		},
	}

	return app
}
