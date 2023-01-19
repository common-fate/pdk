package main

import (
	"os"

	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/pdk/pkg/pdk"
)

func main() {
	app := pdk.GetCliApp()
	err := app.Run(os.Args)
	if err != nil {
		// if the error is an instance of clierr.PrintCLIErrorer then print the error accordingly
		if cliError, ok := err.(clierr.PrintCLIErrorer); ok {
			cliError.PrintCLIError()
		} else {
			clio.Error(err.Error())
		}
		os.Exit(1)
	}
}
