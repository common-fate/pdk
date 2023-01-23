package command

import (
	"encoding/json"
	"os"
)

type Config struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// save the config to JSON file.
func (c *Config) Save(path string) (*os.File, error) {
	cfg, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	_, err = f.Write([]byte(cfg))
	if err != nil {
		return nil, err
	}

	return f, nil
}
