package command

import (
	"net/http"

	"github.com/common-fate/clio"
	"github.com/common-fate/pdk/pkg/cliauth"
	"github.com/common-fate/pdk/pkg/tokenstore"
	"github.com/pkg/browser"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

var Login = cli.Command{
	Name:  "login",
	Usage: "Login to Common Fate Provider Registry",
	Flags: []cli.Flag{},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		authResponse := make(chan cliauth.Response)
		authServer := cliauth.Server{
			Response: authResponse,
		}

		server := &http.Server{
			Addr:    ":8848",
			Handler: authServer.Handler(),
		}

		var g errgroup.Group

		// run the auth server on localhost
		g.Go(func() error {
			clio.Debugw("starting HTTP server", "address", server.Addr)
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				return err
			}
			clio.Debugw("auth server closed")
			return nil
		})

		// open the browser and read the token
		g.Go(func() error {
			url := "http://localhost:8848/oauth/login"
			clio.Infof("Opening your web browser to: %s", url)
			err := browser.OpenURL(url)
			if err != nil {
				clio.Errorf("error opening browser: %s", err)
			}
			return nil
		})

		// read the returned ID token from Cognito
		g.Go(func() error {
			res := <-authResponse

			err := server.Shutdown(ctx)
			if err != nil {
				return err
			}

			// check that the auth flow didn't error out
			if res.Err != nil {
				return err
			}

			ts := tokenstore.New()
			err = ts.Save(res.Token)
			if err != nil {
				return err
			}

			clio.Successf("logged in")

			return nil
		})

		err := g.Wait()
		if err != nil {
			return err
		}

		return nil
	},
}
