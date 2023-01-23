package command

import (
	"encoding/json"

	"github.com/spf13/afero"
)

type Config struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// save the config to JSON file.
func (c *Config) Save(fs afero.Fs, path string) (*afero.File, error) {
	cfg, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	f, err := fs.Create(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	_, err = f.Write([]byte(cfg))
	if err != nil {
		return nil, err
	}

	return &f, nil
}
