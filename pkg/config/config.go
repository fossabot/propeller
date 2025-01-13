package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

type Config struct {
	Manager ManagerConfig `toml:"manager"`
	Proplet PropletConfig `toml:"proplet"`
	Proxy   ProxyConfig   `toml:"proxy"`
}

type ManagerConfig struct {
	ThingID   string `toml:"thing_id"`
	ThingKey  string `toml:"thing_key"`
	ChannelID string `toml:"channel_id"`
}

type PropletConfig struct {
	ThingID   string `toml:"thing_id"`
	ThingKey  string `toml:"thing_key"`
	ChannelID string `toml:"channel_id"`
}

type ProxyConfig struct {
	ThingID   string `toml:"thing_id"`
	ThingKey  string `toml:"thing_key"`
	ChannelID string `toml:"channel_id"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	tree, err := toml.Load(string(data))
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	var config Config
	if err := tree.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}
