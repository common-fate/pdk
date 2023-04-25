package command

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/common-fate/clio"
	"github.com/common-fate/clio/clierr"
	"github.com/urfave/cli/v2"
)

type Config struct {
	Name        string
	Publisher   string
	Version     string
	UseResource bool
}

//go:embed template/**
var templateFiles embed.FS

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

func gitInit(repoDirPath string) error {
	clio.Debugf("git init %s\n", repoDirPath)

	cmd := exec.Command("git", "init", repoDirPath)
	err := cmd.Run()
	if err != nil {
		return err

	}

	return nil
}

func run(ctx *cli.Context, cfg Config, shouldCreateFolder bool) error {
	repoDirPath, err := os.Getwd()
	if err != nil {
		return err
	}

	if shouldCreateFolder {
		repoDirPath = path.Join(repoDirPath, cfg.Name)
		_, err = os.Stat(repoDirPath)
		if err != nil {
			if os.IsNotExist(err) {
				err := os.MkdirAll(repoDirPath, 0777)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		err = gitInit(repoDirPath)
		if err != nil {
			return fmt.Errorf("initializing git repository err: %s", err)
		}
	}

	err = createPyVenv(repoDirPath)
	if err != nil {
		return fmt.Errorf("creating python venv err: %s", err)
	}

	err = fs.WalkDir(templateFiles, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		packageName := cfg.Name
		newPath := ""
		if shouldCreateFolder {
			newPath = path.Join(strings.Replace(p, "template", packageName, 1))
		} else {
			// we need to remove the template portion of the path `template/abc/efg`
			parts := strings.Split(p, "/")
			newPath = path.Join(parts[1:]...)
		}

		if newPath == "" || newPath == "template" {
			return nil
		}

		// If the walked path is directory then create directory and return
		// Subdirectory with `package-name` is replace with provided package name.
		if d.IsDir() {
			_, err := os.Stat(newPath)
			if err != nil {
				if os.IsNotExist(err) {
					clio.Debugf("creating directory %s \n", newPath)
					err := os.Mkdir(newPath, 0777)
					if err != nil {
						return err
					}

					return nil
				}
				return err
			}

			return nil
		}

		f, err := templateFiles.ReadFile(p)
		if err != nil {
			return err
		}

		if !strings.HasSuffix(newPath, ".tmpl") {
			// not a template
			err = os.WriteFile(newPath, f, 0644)
			if err != nil {
				return err
			}
		}

		// if we get here, it's a template that we need to interpolate.
		// an example template file is provider.py.tmpl
		// trim the .tmpl extension - so we're left with provider.py
		newPath = strings.TrimSuffix(newPath, ".tmpl")

		newFile, err := os.Create(newPath)
		if err != nil {
			return err
		}

		defer newFile.Close()

		t, err := template.New("t").Parse(string(f))
		if err != nil {
			return err
		}

		err = t.Execute(newFile, cfg)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	clio.Info("Generating virtual environment for python & installing Packages.")

	err = installPythonDependencies(repoDirPath)
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
			Usage:   "Use template=resource for resource fetching example",
		},
		&cli.StringFlag{
			Name:    "version",
			Aliases: []string{"v"},
			Usage:   "Initial version for the provider",
			Value:   "v0.1.0",
		},
		&cli.BoolFlag{
			Name:  "create-folder",
			Usage: "Create a new folder and initialize a git repository",
		},
	},
	Action: func(c *cli.Context) error {
		name := c.String("name")
		publisher := c.String("publisher")
		version := c.String("version")
		shouldCreateFolder := c.Bool("create-folder")

		useResource := false
		if c.String("template") == "resource" {
			clio.Info("Scaffolding a resource fetching boilerplate example")
			useResource = true
		}

		files, err := os.ReadDir(".")
		if err != nil {
			log.Fatal(err)
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

		config := Config{
			Name:        name,
			Publisher:   publisher,
			Version:     version,
			UseResource: useResource,
		}

		err = run(c, config, shouldCreateFolder)
		if err != nil {
			return err
		}

		clio.Success("Success! Scaffolded a new Common Fate Provider")
		clio.Info("Get started by running these commands next:")
		fmt.Println("source .venv/bin/activate")
		fmt.Println("pdk test describe")
		return nil
	},
}

// InstallPythonDependencies looks for the generated venv path
// and installs commonfate_provider package and other dependencies packages.
// then it creates requirements.txt file based on the output of pip freeze command.
func installPythonDependencies(p string) error {
	clio.Info("running .venv/bin/pip install commonfate_provider black")

	cmd := exec.Command(".venv/bin/pip", "install", "commonfate_provider", "black")

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
