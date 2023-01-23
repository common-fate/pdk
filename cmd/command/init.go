package command

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/common-fate/clio"
	"github.com/urfave/cli/v2"
)

//go:embed template/**
var templateFiles embed.FS

func run(ctx *cli.Context, repoDirPath string, cfg Config) error {
	_, err := os.Stat(repoDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(repoDirPath, 0777)
			if err != nil {
				return err
			}

		} else {
			return err
		}
	}

	err = gitInit(repoDirPath)
	if err != nil {
		return nil
	}

	err = fs.WalkDir(templateFiles, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		packageName := cfg.Name
		newPath := strings.Replace(p, "template", packageName, 1)
		newPath = strings.Replace(newPath, "provider", packageName, 1)
		filename := path.Base(p)

		// If the walked path is directory then create directory and return
		// Subdirectory with `package-name` is replace with provided package name.
		if d.IsDir() {
			_, err := os.Stat(newPath)
			if err != nil {
				if os.IsNotExist(err) {
					// replace package-name directory name with provided provider registry name.
					if filename == "provider" {
						newPath = strings.Replace(newPath, "provider", cfg.Name, -1)
					}

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

		fileExtension := ".py"
		if filename == "pyproject.tmpl" {
			fileExtension = ".toml"
		}

		newFile, err := os.Create(path.Join(strings.Replace(newPath, ".tmpl", fileExtension, 1)))
		if err != nil {
			fmt.Println("err", err.Error())
			return err
		}

		defer newFile.Close()

		// if the file currently if of type .tmpl then interpolate with go template.
		// else copy the content as it is.
		if filepath.Ext(newPath) == ".tmpl" {
			t := template.Must(template.New("t").Parse(string(f)))
			err = t.Execute(newFile, cfg)
			if err != nil {
				panic(err)
			}
		} else {
			_, err = newFile.Write(f)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

var Init = cli.Command{
	Name:  "init",
	Usage: "",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Required: true,
			Aliases:  []string{"n"},
			Usage:    "name of the package",
		},
		&cli.StringFlag{
			Name:    "description",
			Aliases: []string{"d"},
			Usage:   "description of the package",
		},
	},
	Action: func(c *cli.Context) error {
		name := c.String("name")
		description := c.String("description")

		config := Config{
			Name:        name,
			Description: description,
		}

		repoDirPath := config.Name

		err := run(c, repoDirPath, config)
		if err != nil {
			clio.Debugf("Failed to scaffold with error %s", err)

			// // remove all the files if there is error.
			// err := AppFs.RemoveAll(repoDirPath)
			// if err != nil {
			// 	return err
			// }

			return err
		}

		_, err = config.Save(path.Join(repoDirPath, "config.json"))
		if err != nil {
			return err
		}

		clio.Success("Successfully scaffolded a new Provider repository")
		return nil
	},
}
