package run

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"

	"github.com/common-fate/clio"
	"github.com/common-fate/provider-registry-sdk-go/pkg/msg"
)

type payload struct {
	Type msg.RequestType `json:"type"`
	Data any             `json:"data"`
}

func RunEntrypoint(event msg.Request, env map[string]string) (*msg.Result, error) {
	payload := payload{
		Type: event.Type(),
		Data: event,
	}

	eventBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	// This will escape special characters and wrap the josn in quotes "{\"a\":1}"
	// so that it can be used as an argument to the entrypoint script
	eventString := string(eventBytes)
	clio.Debugw("running provider", "event", eventString)

	var b bytes.Buffer

	cmd := exec.Command(".venv/bin/provider", "run", eventString)
	cmd.Stdout = &b
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	// forward the env of the caller to the script
	// this means AWS creds in the env will be available to the python code etc
	cmd.Env = append(cmd.Env, os.Environ()...)
	if env != nil {
		b, err := json.Marshal(env)
		if err != nil {
			return nil, err
		}
		cmd.Env = append(cmd.Env, "PROVIDER_CONFIG="+string(b))
	}

	err = cmd.Run()
	if err != nil {
		return nil, err
	}

	clio.Debugf("out: %s", b.String())

	var res msg.Result
	dec := json.NewDecoder(&b)

	// use stricter decoding to try and catch invalid responses from the provider here.
	dec.DisallowUnknownFields()

	err = dec.Decode(&res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}
