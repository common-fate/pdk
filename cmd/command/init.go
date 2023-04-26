package command

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/boilermaker"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/common-fate/pdk/boilerplate"
	"github.com/urfave/cli/v2"
)

type TemplateData struct {
	// PackageName is the name of the Python package folder to create
	PackageName string
	Name        string
	Publisher   string
	Version     string
}

// getPythonCommand finds the actual command to run Python.
// It returns an error if we can't find a Python executable in
// the system path.
func getPythonCommand() (string, error) {
	commands := []string{"python3", "python3.9", "python"}

	for _, cmd := range commands {
		_, err := exec.LookPath(cmd)
		if err == nil {
			return cmd, nil
		}
	}

	msg := fmt.Sprintf("Python is required to develop Common Fate Providers, but we couldn't find it in your system path (we checked for %s). Please install Python to continue.", strings.Join(commands, ", "))
	return "", clierr.New(msg)
}

func createPyVenv(p string) error {
	py, err := getPythonCommand()
	if err != nil {
		return err
	}
	cmd := exec.Command(py, "-m", "venv", ".venv")
	cmd.Dir = path.Join(p)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

var Init = cli.Command{
	Name:  "init",
	Usage: "Scaffold a template for an Access Provider",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "name",
			Aliases: []string{"n"},
			Usage:   "Provider name",
		},
		&cli.StringFlag{
			Name:    "publisher",
			Aliases: []string{"p"},
			Usage:   "Provider publisher",
		},
		&cli.StringFlag{
			Name:    "template",
			Aliases: []string{"t"},
			Usage:   "The Provider template to use",
			Value:   "basic",
		},
		&cli.StringFlag{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Initial version for the Provider",
			Value:   "v0.1.0",
		},
		&cli.BoolFlag{
			Name:  "create-folder",
			Usage: "Create a new folder for the Provider",
		},
	},
	Action: func(c *cli.Context) error {
		name := c.String("name")
		publisher := c.String("publisher")
		version := c.String("version")
		shouldCreateFolder := c.Bool("create-folder")

		files, err := os.ReadDir(".")
		if err != nil {
			return err
		}

		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		// if the directory contains any file
		if len(files) != 0 && !shouldCreateFolder {
			clio.Errorf("you are running `pdk init` in a non-empty directory! %s", dir)
			clio.Info("You need to pass `pdk init --create-folder` to create a new Provider repository")

			return nil
		}

		if name == "" {
			in := &survey.Input{
				Message: "Provider name",
				Default: strings.TrimPrefix(path.Base(dir), "cf-provider-"),
			}
			err = survey.AskOne(in, &name)
			if err != nil {
				return err
			}
		}

		if publisher == "" {
			in := &survey.Input{
				Message: "Provider publisher",
				Help:    "This should match the GitHub organization, or your personal GitHub name",
			}
			err = survey.AskOne(in, &publisher)
			if err != nil {
				return err
			}
		}

		templateFlag := c.String("template")

		data := TemplateData{
			PackageName: "provider_" + strings.ReplaceAll(name, "-", "_"),
			Name:        name,
			Publisher:   publisher,
			Version:     version,
		}

		boilerplates, err := boilermaker.ParseMapFS(boilerplate.TemplateFiles, "templates")
		if err != nil {
			return err
		}

		boilerplate, ok := boilerplates[templateFlag]
		if !ok {
			var availableTemplates []string

			for k := range boilerplates {
				availableTemplates = append(availableTemplates, k)
			}

			return fmt.Errorf("invalid template %s. available templates: %s", templateFlag, strings.Join(availableTemplates, ", "))
		}

		result, err := boilerplate.Generate(data)
		if err != nil {
			return err
		}

		if shouldCreateFolder {
			dir = path.Join(dir, "cf-provider-"+data.Name)
			_, err = os.Stat(dir)
			if err != nil {
				if os.IsNotExist(err) {
					err := os.MkdirAll(dir, 0777)
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}
		}

		for f, contents := range result {
			fullpath := filepath.Join(dir, f)
			parent := filepath.Dir(fullpath)
			err := os.MkdirAll(parent, 0755)
			if err != nil {
				return err
			}

			err = os.WriteFile(fullpath, []byte(contents), 0644)
			if err != nil {
				return err
			}
			clio.Infof("created %s", fullpath)
		}

		err = createPyVenv(dir)
		if err != nil {
			return fmt.Errorf("creating python venv err: %s", err)
		}

		clio.Info("Generating virtual environment for python & installing Packages.")

		err = installPythonDependencies(dir)
		if err != nil {
			return err
		}

		clio.Success("Success! Scaffolded a new Common Fate Provider")
		clio.Info("Get started by running these commands next:")
		fmt.Println("source .venv/bin/activate")
		fmt.Println("pdk run describe")
		return nil
	},
}

// InstallPythonDependencies looks for the generated venv path
// and installs commonfate_provider package and other dependencies packages.
// then it creates requirements.txt file based on the output of pip freeze command.
func installPythonDependencies(p string) error {
	clio.Info("running .venv/bin/pip install provider black structlog")

	cmd := exec.Command(".venv/bin/pip", "install", "provider", "black", "structlog")

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return err
	}

	clio.Info("running .venv/bin/pip freeze > requirements.txt")

	var b bytes.Buffer
	cmd = exec.Command(".venv/bin/pip", "freeze")
	cmd.Stderr = os.Stderr
	cmd.Stdout = &b
	err = cmd.Run()
	if err != nil {
		return err
	}

	err = os.WriteFile("requirements.txt", b.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}
