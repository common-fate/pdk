package resources

import (
	"encoding/json"
	"fmt"

	"github.com/common-fate/pdk/cmd/run"
	"github.com/common-fate/provider-registry-sdk-go/pkg/msg"
	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

var Command = cli.Command{
	Name: "resources",
	Subcommands: []*cli.Command{
		&taskCommand,
		&loadCommand,
	},
}

var taskCommand = cli.Command{
	Name:        "task",
	Subcommands: []*cli.Command{&runTaskCommand},
}

var runTaskCommand = cli.Command{
	Name: "run",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "task", Required: true},
		&cli.StringFlag{Name: "ctx", Value: "{}"},
	},
	Action: func(c *cli.Context) error {
		env, _ := godotenv.Read()
		var contx map[string]any
		err := json.Unmarshal([]byte(c.String("ctx")), &contx)
		if err != nil {
			return err
		}
		task := c.String("task")
		out, err := run.RunEntrypoint(msg.LoadResources{Task: task, Ctx: contx}, env)
		if err != nil {
			return err
		}
		fmt.Println(string(out.Response))
		return nil
	},
}

var loadCommand = cli.Command{
	Name: "load",
	Action: func(c *cli.Context) error {
		env, _ := godotenv.Read()

		out, err := run.RunEntrypoint(msg.Describe{}, env)
		if err != nil {
			return err
		}

		var describe providerregistrysdk.DescribeResponse
		err = json.Unmarshal(out.Response, &describe)
		if err != nil {
			return err
		}
		//then call the local version of fetch resources
		rf := ResourceFetcher{eg: &errgroup.Group{}}

		var tasks []string
		for key := range describe.Schema.Resources.Loaders {
			tasks = append(tasks, key)
		}

		return rf.LoadResources(c.Context, tasks)
	},
}
