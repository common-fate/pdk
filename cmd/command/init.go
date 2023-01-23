package command

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/common-fate/clio"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v2"
)

func run(ctx *cli.Context, AppFs afero.Fs, repoDirPath string, cfg Config) error {
	_, err := AppFs.Stat(repoDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			err := AppFs.Mkdir(repoDirPath, 0777)
			if err != nil {
				return err
			}

		} else {
			return err
		}
	} else {
		err = AppFs.RemoveAll(repoDirPath)
		if err != nil {
			return err
		}
	}

	err = AppFs.Mkdir(repoDirPath, 0777)
	if err != nil {
		return err
	}

	err = gitInit(repoDirPath)
	if err != nil {
		return nil
	}

	err = afero.Walk(AppFs, "./template", func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		packageName := cfg.Name
		newPath := strings.Replace(p, "template", packageName, 1)
		newPath = strings.Replace(newPath, "package-name", packageName, 1)
		filename := path.Base(p)

		isDir, err := afero.IsDir(AppFs, p)
		if err != nil {
			return err
		}

		// If the walked path is directory then create directory and return
		// Subdirectory with `package-name` is replace with provided package name.
		if isDir {
			_, err := AppFs.Stat(newPath)
			if err != nil {
				if os.IsNotExist(err) {
					// replace package-name directory name with provided provider registry name.
					if filename == "package-name" {
						newPath = strings.Replace(newPath, "package-name", cfg.Name, -1)
					}

					clio.Debugf("creating directory %s \n", newPath)
					err := AppFs.Mkdir(newPath, 0777)
					if err != nil {
						return err
					}
				}
			}

			return nil
		}

		f, err := afero.ReadFile(AppFs, p)
		if err != nil {
			return err
		}

		fileExtension := ".py"
		if filename == "pyproject.tmpl" {
			fileExtension = ".toml"
		}

		newFile, err := AppFs.Create(path.Join(strings.Replace(newPath, ".tmpl", fileExtension, 1)))
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
		var AppFs = afero.NewOsFs()

		name := c.String("name")
		description := c.String("description")

		config := Config{
			Name:        name,
			Description: description,
		}

		_, err := config.Save(AppFs)
		if err != nil {
			return err
		}

		repoDirPath := config.Name

		err = run(c, AppFs, repoDirPath, config)
		if err != nil {
			clio.Debugf("Failed to scaffold with error %s", err)

			// remove all the files if there is error.
			err := AppFs.RemoveAll(repoDirPath)
			if err != nil {
				return err
			}

			return err
		}

		clio.Success("Successfully scaffolded a new Provider repository")
		return nil
	},
}
