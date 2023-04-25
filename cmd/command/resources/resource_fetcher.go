package resources

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/common-fate/clio"
	"github.com/common-fate/pdk/cmd/run"
	"github.com/common-fate/provider-registry-sdk-go/pkg/msg"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

// ResourceFetcher fetches resources from provider lambda handler based on
// provider schema's "loadResources" object.
type ResourceFetcher struct {
	eg *errgroup.Group
}

// LoadResources invokes the deployment
func (rf *ResourceFetcher) LoadResources(ctx context.Context, tasks []string) error {
	for _, task := range tasks {
		// copy the loop variable
		tc := task

		err := runTasksRecursive(tc, map[string]any{})
		if err != nil {
			return err
		}
	}
	return nil
}

func runTasksRecursive(name string, ctx map[string]any) error {
	clio.Debugw("running task", "name", name, "ctx", ctx)
	res, err := runTask(name, ctx)
	if err != nil {
		return err
	}

	clio.Debugw("got task results", "result", res)

	for _, resource := range res.Resources {
		clio.Infow("Found Resource", "resource", resource)
	}

	for _, subtask := range res.Tasks {
		err = runTasksRecursive(subtask.Task, subtask.Ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// runTask calls a resource loading function in the provider.
// Note that the provided ctx variable is NOT a context.Context Go context, but is
// rather a dict of arguments to be provided to the loader function in Python.
func runTask(name string, ctx map[string]any) (*msg.LoadResponse, error) {
	env, err := godotenv.Read()
	if err != nil {
		return nil, err
	}
	out, err := run.RunEntrypoint(msg.LoadResources{Task: name, Ctx: ctx}, env)
	if err != nil {
		return nil, err
	}

	var v msg.LoadResponse
	dec := json.NewDecoder(bytes.NewBuffer(out.Response))

	// use stricter decoding to try and catch invalid responses from the provider here.
	// dec.DisallowUnknownFields()
	err = dec.Decode(&v)
	if err != nil {
		return nil, err
	}

	return &v, nil
}
