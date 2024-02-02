package config

import (
    "fmt"
    "gopkg.in/yaml.v2"
    "io/ioutil"
)

var terraformDriver string
var terraformDirArg string

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
  	TerraformScheduling string `yaml:"terraform_scheduling,omitempty"`
}

func RenderConfig(config *InitConfig, ymlConfigPath string) error {
    yamlConfig := YAMLConfig{
        ListenAddress: config.Server.Listen,
        Verbose:       config.Server.LogLevel == "info",
    }

    for _, task := range config.Tasks {
        if task.Condition.Labels != nil {
            terraformDriver := config.Server.TerraformDriver
            if task.TerraformDriver != "" {
              terraformDriver = task.TerraformDriver
            }
            switch {
            case terraformDriver == "terraform":
              terraformDirArg = "-chdir="
            case terraformDriver == "terragrunt":
              terraformDirArg = "--terragrunt-working-dir "
            }
            chDir := fmt.Sprintf("%s%s", terraformDirArg, task.Source)
            taskCmd := []string{chDir, "apply", "-auto-approve"}
            cmd := InitCommand{
                Cmd:              terraformDriver,
                Args:             taskCmd,
                MatchLabels:      task.Condition.Labels,
                NotifyOnFailure:  task.Condition.NotifyOnFailure,
                ResolvedSignal:   task.Condition.ResolvedSignal,
                IgnoreResolved:   task.Condition.IgnoreResolved,
                TerraformScheduling:   task.Condition.TerraformScheduling,
                Max:              1,
            }
            yamlConfig.Commands = append(yamlConfig.Commands, cmd)
        }
    }

    data, err := yaml.Marshal(&yamlConfig)
    if err != nil {
        return err
    }
    return ioutil.WriteFile(ymlConfigPath, data, 0644)
}
