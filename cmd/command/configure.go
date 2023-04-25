package command

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

var Configure = cli.Command{
	Name:  "configure",
	Usage: "Update or create .env file with all the required configuration fields",
	Flags: []cli.Flag{},
	Action: func(c *cli.Context) error {
		var out bytes.Buffer
		cmd := exec.Command(".venv/bin/provider", "schema")
		cmd.Stderr = os.Stderr
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			return err
		}

		var schema providerregistrysdk.Schema
		err = json.Unmarshal(out.Bytes(), &schema)
		if err != nil {
			return err
		}

		values := make(map[string]string)

		if schema.Config == nil {
			// provider has no config schema
			return nil
		}

		for k, v := range *schema.Config {
			// prompt the user for each config value
			var ans string
			err = survey.AskOne(&survey.Input{Message: k + ":"}, &ans)
			if err != nil {
				return err
			}

			if v.Secret != nil && *v.Secret {
				values["PROVIDER_SECRET_"+strings.ToUpper(k)] = ans
			} else {
				values["PROVIDER_CONFIG_"+strings.ToUpper(k)] = ans
			}
		}

		err = godotenv.Write(values, ".env")
		if err != nil {
			return err
		}

		return nil
	},
}
