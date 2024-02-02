package config

import (
    "fmt"
    "github.com/hashicorp/hcl/v2"
    "github.com/hashicorp/hcl/v2/hclparse"
    "github.com/zclconf/go-cty/cty"
	log "github.com/cloudputation/iterator/packages/logger"
)

type InitConfig struct {
    Server *Server
    Tasks  []*Task
}

type Server struct {
    DataDir string
    LogDir string
    LogLevel string
    Listen string
    Address string
    TerraformDriver string
    Consul          ConsulConfig
}

type ConsulConfig struct {
    Address string
}

type Task struct {
    Name        string
    Description string
    Source      string
    TerraformDriver string 
    Condition   Condition
}

type Condition struct {
    TerraformScheduling string
    NotifyOnFailure bool
    ResolvedSignal  string
    IgnoreResolved  bool
    Labels          map[string]string
}

const (
  defaultListenAddr = "9595"
)

var ConsulFactoryDataDir = "iterator::Data"
var ConsulStorageEnabled bool

func LoadConfig(configPath string) (*InitConfig, error) {
  config := &InitConfig{}

  parser := hclparse.NewParser()
  file, diags := parser.ParseHCLFile(configPath)
  if diags.HasErrors() {
      return nil, fmt.Errorf("failed to parse HCL file: %s", diags)
  }

  content, diags := file.Body.Content(&hcl.BodySchema{
      Blocks: []hcl.BlockHeaderSchema{
          {Type: "task"},
          {Type: "server"},
      },
  })
  if diags.HasErrors() {
      return nil, fmt.Errorf("failed to get file content: %s", diags)
  }


  for _, block := range content.Blocks {
      switch block.Type {
      case "task":
          taskMap, err := processTaskBlock(block)
          if err != nil {
            return nil, fmt.Errorf("failed to process task block %w", err)
          }
          if taskMap != nil {
              task := populateTaskStruct(taskMap)
              config.Tasks = append(config.Tasks, task)
          }
      case "server":
          serverMap, err := processServerBlock(block)
          if err != nil {
            return nil, fmt.Errorf("failed to process server block %w", err)
          }
          if serverMap != nil {
              config.Server = populateServerStruct(serverMap)
          }
      }
  }

  return config, nil
}

func processServerBlock(serverBlock *hcl.Block) (map[string]interface{}, error) {
  serverData := make(map[string]interface{})

  content, _, diags := serverBlock.Body.PartialContent(&hcl.BodySchema{
      Attributes: []hcl.AttributeSchema{
          {Name: "data_dir"},
          {Name: "log_level"},
          {Name: "log_dir"},
          {Name: "listen"},
          {Name: "address"},
          {Name: "terraform_driver"},
      },
      Blocks: []hcl.BlockHeaderSchema{
          {Type: "consul"},
      },
  })
  if diags.HasErrors() {
      return nil, fmt.Errorf("failed to get server content: %s", diags)
  }

  for k, attr := range content.Attributes {
      val, diags := attr.Expr.Value(nil)
      if diags.HasErrors() {
          log.Error("Failed to decode attribute value for %s: %s", k, diags)
          continue
      }
      serverData[k] = val.AsString()
  }

  for _, block := range content.Blocks {
      if block.Type == "consul" {
          consulData, err := processConsulBlock(block)
          if err != nil {
            return nil, fmt.Errorf("failed to process Consul block %w", err)
          }
          if consulData != nil {
              serverData["consul"] = consulData
              ConsulStorageEnabled = true
          }
      }
  }

  return serverData, nil
}

func processConsulBlock(consulBlock *hcl.Block) (map[string]interface{}, error) {
  consulData := make(map[string]interface{})

  attrs, diags := consulBlock.Body.JustAttributes()
  if diags.HasErrors() {
      return nil, fmt.Errorf("failed to decode consul attributes: %s", diags)
  }

  for key, attr := range attrs {
      val, diags := attr.Expr.Value(nil)
      if diags.HasErrors() {
          log.Error("Failed to decode attribute value for %s: %s", key, diags)
          continue
      }
      consulData[key] = val.AsString()
  }

  return consulData, nil
}

func populateServerStruct(serverMap map[string]interface{}) *Server {
  server := &Server{
      DataDir: serverMap["data_dir"].(string),
      LogLevel: serverMap["log_level"].(string),
      LogDir: serverMap["log_dir"].(string),
      Listen: defaultListenAddr,
      Address: serverMap["address"].(string),
      TerraformDriver: serverMap["terraform_driver"].(string),
  }

  if listen, ok := serverMap["listen"]; ok {
      server.Listen = listen.(string)
  }

  if consul, ok := serverMap["consul"].(map[string]interface{}); ok {
      server.Consul = ConsulConfig{
          Address: consul["address"].(string),
      }
  }

  return server
}

func processTaskBlock(taskBlock *hcl.Block) (map[string]interface{}, error) {
  taskData := make(map[string]interface{})

  content, _, diags := taskBlock.Body.PartialContent(&hcl.BodySchema{
      Attributes: []hcl.AttributeSchema{
          {Name: "name"},
          {Name: "description"},
          {Name: "source"},
          {Name: "terraform_driver"},
      },
      Blocks: []hcl.BlockHeaderSchema{
          {Type: "condition", LabelNames: []string{"type"}},
      },
  })
  if diags.HasErrors() {
      return nil, fmt.Errorf("failed to get task content %s", diags)
  }

  for k, attr := range content.Attributes {
      val, diags := attr.Expr.Value(nil)
      if diags.HasErrors() {
          log.Error("Failed to decode attribute value for %s: %s", k, diags)
          continue
      }
      taskData[k] = val.AsString()
  }

  for _, block := range content.Blocks {
      if block.Type == "condition" && len(block.Labels) > 0 && block.Labels[0] == "label-match" {
          conditionData, err := processConditionBlock(block)
          if err != nil {
            return nil, fmt.Errorf("failed to process condition block %w", err)
          }
          if conditionData != nil {
              taskData["condition"] = conditionData
          }
      }
  }

  return taskData, nil
}

func processConditionBlock(conditionBlock *hcl.Block) (map[string]interface{}, error) {
  conditionData := make(map[string]interface{})

  content, _, diags := conditionBlock.Body.PartialContent(&hcl.BodySchema{
      Attributes: []hcl.AttributeSchema{
          {Name: "notify_on_failure"},
          {Name: "terraform_scheduling"},
          {Name: "resolved_signal"},
          {Name: "ignore_resolved"},
      },
      Blocks: []hcl.BlockHeaderSchema{{Type: "label"}},
  })
  if diags.HasErrors() {
      return nil, fmt.Errorf("failed to get condition content %s", diags)
  }

  for k, attr := range content.Attributes {
      val, diags := attr.Expr.Value(nil)
      if diags.HasErrors() {
          log.Error("Failed to decode attribute value for %s: %s", k, diags)
          continue
      }
      if val.Type().Equals(cty.Bool) {
          conditionData[k] = val.True()
      } else {
          conditionData[k] = val.AsString()
      }
  }

  for _, labelBlock := range content.Blocks {
      if labelBlock.Type == "label" {
          labelData, err := processLabelBlock(labelBlock)
          if err != nil {
            return nil, fmt.Errorf("failed to process label block %w", err)
          }
          if labelData != nil {
              conditionData["labels"] = labelData
          }
      }
  }

  return conditionData, nil
}

func processLabelBlock(labelBlock *hcl.Block) (map[string]string, error) {
  labels := make(map[string]string)

  attrs, diags := labelBlock.Body.JustAttributes()
  if diags.HasErrors() {
      return nil, fmt.Errorf("failed to decode label attributes %s", diags)
  }

  for key, attr := range attrs {
      val, diags := attr.Expr.Value(nil)
      if diags.HasErrors() {
          log.Error("Failed to decode attribute value for %s: %s", key, diags)
          continue
      }
      labels[key] = val.AsString()
  }

  return labels, nil
}

func populateTaskStruct(taskMap map[string]interface{}) *Task {
  task := &Task{
      Name:        taskMap["name"].(string),
      Description: taskMap["description"].(string),
      Source:      taskMap["source"].(string),
  }

  if terraformDriver, ok := taskMap["terraform_driver"]; ok {
      task.TerraformDriver = terraformDriver.(string)
  }

  if cond, ok := taskMap["condition"].(map[string]interface{}); ok {
      task.Condition = populateConditionStruct(cond)
  }

  return task
}

func populateConditionStruct(condMap map[string]interface{}) Condition {
  condition := Condition{
      NotifyOnFailure: condMap["notify_on_failure"].(bool),
      ResolvedSignal:  condMap["resolved_signal"].(string),
      IgnoreResolved:  condMap["ignore_resolved"].(bool),
  }

  if terraformScheduling, ok := condMap["terraform_scheduling"]; ok {
      condition.TerraformScheduling = terraformScheduling.(string)
  }

  if labels, ok := condMap["labels"].(map[string]string); ok {
      condition.Labels = labels
  }

  return condition
}
