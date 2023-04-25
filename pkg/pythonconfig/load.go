package pythonconfig

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/common-fate/provider-registry-sdk-go/pkg/providerregistrysdk"
)

type MetaInfo struct {
	Icon        string `toml:"icon"`
	DisplayName string `toml:"displayName"`
	Source      string `toml:"source"`
}

func (m MetaInfo) ToAPI() providerregistrysdk.ProviderMetaInfo {
	me := providerregistrysdk.ProviderMetaInfo{}
	if m.Icon != "" {
		me.Icon = &m.Icon
	}
	if m.DisplayName != "" {
		me.DisplayName = &m.DisplayName
	}
	if m.Source != "" {
		me.Source = &m.Source
	}
	return me
}

type Config struct {
	Name      string   `toml:"name"`
	Publisher string   `toml:"publisher"`
	Version   string   `toml:"version"`
	Language  string   `toml:"language"`
	Meta      MetaInfo `toml:"meta"`
}

func LoadFile(filepath string) (Config, error) {
	var cfg Config

	f, err := os.Open(filepath)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	dec := toml.NewDecoder(f)
	_, err = dec.Decode(&cfg)
	return cfg, err
}
