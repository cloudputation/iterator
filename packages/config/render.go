package config

import (
    "fmt"
    "gopkg.in/yaml.v2"
    "io/ioutil"
)

type YAMLConfig struct {
    ListenAddress string        `yaml:"listen_address"`
    Verbose       bool          `yaml:"verbose"`
    TLSKey        string        `yaml:"tls_key,omitempty"`
    TLSCrt        string        `yaml:"tls_crt,omitempty"`
    Commands      []InitCommand `yaml:"commands"`
}

type InitCommand struct {
    Cmd             string            `yaml:"cmd"`
    Args            []string          `yaml:"args,omitempty"`
    MatchLabels     map[string]string `yaml:"match_labels,omitempty"`
    NotifyOnFailure bool              `yaml:"notify_on_failure"`
    ResolvedSignal  string            `yaml:"resolved_signal,omitempty"`
    IgnoreResolved  bool              `yaml:"ignore_resolved,omitempty"`
    Max             int               `yaml:"max,omitempty"`
}

func RenderConfig(config *InitConfig) error {
    yamlConfig := YAMLConfig{
        ListenAddress: config.Server.Listen,
        Verbose:       config.Server.LogLevel == "DEBUG",
    }

    terraformDriver := config.Server.TerraformDriver

    for _, task := range config.Tasks {
        if task.Condition.Labels != nil {
            chDir := fmt.Sprintf("-chdir=%s", task.Source)
            taskCmd := []string{chDir, "apply", "-auto-approve"}
            cmd := InitCommand{
                Cmd:              terraformDriver,
                Args:             taskCmd,
                MatchLabels:      task.Condition.Labels,
                NotifyOnFailure:  task.Condition.NotifyOnFailure,
                ResolvedSignal:   task.Condition.ResolvedSignal,
                IgnoreResolved:   task.Condition.IgnoreResolved,
                Max:              1,
            }
            yamlConfig.Commands = append(yamlConfig.Commands, cmd)
        }
    }

    data, err := yaml.Marshal(&yamlConfig)
    if err != nil {
        return err
    }
    return ioutil.WriteFile("config.yml", data, 0644)
}
