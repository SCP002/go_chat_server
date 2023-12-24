package config

import (
	"os"

	"github.com/cockroachdb/errors"
	"github.com/pelletier/go-toml/v2"
)

const configFileName = "go_chat_server_config.toml"

// Config represents config file contents.
type Config struct {
	ListenAddress string `toml:"listen_address" comment:"Address to listen to in format of 'host:port' or ':port'"`
}

// Read reads and returns config file.
func Read() (Config, error) {
	bytes, err := os.ReadFile(configFileName)
	if err != nil {
		return Config{}, errors.Wrap(err, "Read config file")
	}

	var cfg Config
	err = toml.Unmarshal(bytes, &cfg)
	if err != nil {
		return Config{}, errors.Wrap(err, "Decode config file")
	}

	return cfg, nil
}

// Write writes <cfg> to file.
func Write(cfg Config) error {
	bytes, err := toml.Marshal(cfg)
	if err != nil {
		return errors.Wrap(err, "Encode config file")
	}

	err = os.WriteFile(configFileName, bytes, 0644)
	return errors.Wrap(err, "Write config file")
}
