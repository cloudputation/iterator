package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"

	"github.com/cloudputation/iterator/packages/command"
	log "github.com/cloudputation/iterator/packages/logger"
)

const (
	defaultListenAddr = "9595"
)

// Config represents the configuration for this program
type Config struct {
	ListenAddr string     `yaml:"listen_address"`
	Verbose    bool       `yaml:"verbose"`
	TLSKey     string     `yaml:"tls_key"`
	TLSCrt     string     `yaml:"tls_crt"`
	Commands   []*command.Command `yaml:"commands"`
}

// HasCommand returns true if the config contains the given Command
func (c *Config) HasCommand(other *command.Command) bool {
	for _, cmd := range c.Commands {
		if cmd.Equal(other) {
			return true
		}
	}
	return false
}

// mergeConfigs returns a config representing all the Configs merged together,
// with later Config structs overriding settings in earlier ones (like ListenAddr).
// Commands are added if they are unique from others.
func mergeConfigs(all ...*Config) *Config {
	var merged = &Config{}

	for _, c := range all {
		if c == nil {
			continue
		}

		if len(c.ListenAddr) > 0 {
			merged.ListenAddr = c.ListenAddr
		}
		merged.Verbose = merged.Verbose || c.Verbose
		if c.TLSKey != "" {
			merged.TLSKey = c.TLSKey
		}
		if c.TLSCrt != "" {
			merged.TLSCrt = c.TLSCrt
		}

		for _, cmd := range c.Commands {
			if !merged.HasCommand(cmd) {
				merged.Commands = append(merged.Commands, cmd)
			}
		}
	}

	return merged
}


// readConfig reads configuration from supported means (cli flags, config file),
// validates parameters and returns a Config struct.
func ReadConfig(configFile string) (*Config, error) {
    // Initialize a default configuration
    c := &Config{ListenAddr: defaultListenAddr}

    // Read configuration from a file if provided
    if len(configFile) > 0 {
        fileConfig, err := readConfigFile(configFile)
        if err != nil {
            return nil, fmt.Errorf("error reading config file: %w", err)
        }
        c = mergeConfigs(c, fileConfig) // Merge file configuration with default
    }

    // Validate the commands in the merged configuration
    for i, cmd := range c.Commands {
        _, err := cmd.ParseSignal()
        if err != nil {
            return nil, fmt.Errorf("invalid resolved_signal specified for command %q at index %d: %w", cmd, i, err)
        }

        if cmd.IgnoreResolved != nil && *cmd.IgnoreResolved {
            log.Warn("Warning: command %q at index %d specifies a resolved_signal, and also specifies to ignore resolved alert. The signal won't be used.", cmd, i)
        }
    }

    // Ensure at least one command is configured
    if len(c.Commands) == 0 {
        return nil, fmt.Errorf("missing command to execute on receipt of alarm")
    }

    return c, nil
}

// readConfigFile reads configuration from a yaml file
func readConfigFile(name string) (*Config, error) {
    var c Config
    data, err := ioutil.ReadFile(name)
    if err != nil {
        return nil, err
    }

    err = yaml.Unmarshal(data, &c)
    return &c, err
}
