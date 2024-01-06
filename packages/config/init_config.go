package config

import (
    "fmt"
    "github.com/hashicorp/hcl/v2"
    "github.com/hashicorp/hcl/v2/hclparse"
    "github.com/zclconf/go-cty/cty"
)

type InitConfig struct {
    Server *Server
    Tasks  []*Task
}

type Server struct {
    Listen string
    DataDir string
    LogLevel string
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
    Condition   Condition
}

type Condition struct {
    NotifyOnFailure bool
    ResolvedSignal  string
    IgnoreResolved  bool
    Labels          map[string]string
}

// Init Config prepares the server by loading a preconfiguratin file that will be used
// to generate the executor's config as well as other current and future server parameters
func printConfig(config *InitConfig) {
    if config.Server != nil {
        fmt.Printf("Server:\n")
        fmt.Printf("  Listen: %s\n", config.Server.Listen)
        fmt.Printf("  DataDirectory: %s\n", config.Server.DataDir)
        fmt.Printf("  LogLevel: %s\n", config.Server.LogLevel)
        fmt.Printf("  Terraform Driver: %s\n", config.Server.TerraformDriver)
        if config.Server.Consul.Address != "" {
            fmt.Printf("  Consul:\n")
            fmt.Printf("    Address: %s\n", config.Server.Consul.Address)
        }
        fmt.Println()
    }

    for _, task := range config.Tasks {
        fmt.Printf("Task:\n")
        fmt.Printf("  Name: %s\n", task.Name)
        fmt.Printf("  Description: %s\n", task.Description)
        fmt.Printf("  Source: %s\n", task.Source)
        fmt.Printf("  Condition:\n")
        fmt.Printf("    NotifyOnFailure: %v\n", task.Condition.NotifyOnFailure)
        fmt.Printf("    ResolvedSignal: %s\n", task.Condition.ResolvedSignal)
        fmt.Printf("    IgnoreResolved: %v\n", task.Condition.IgnoreResolved)
        fmt.Printf("    Labels:\n")
        for key, value := range task.Condition.Labels {
            fmt.Printf("      %s: %s\n", key, value)
        }
        fmt.Println()
    }
}

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
            taskMap := processTaskBlock(block)
            if taskMap != nil {
                task := populateTaskStruct(taskMap)
                config.Tasks = append(config.Tasks, task)
            }
        case "server":
            serverMap := processServerBlock(block)
            if serverMap != nil {
                config.Server = populateServerStruct(serverMap)
            }
        }
    }

    return config, nil
}

func processServerBlock(serverBlock *hcl.Block) map[string]interface{} {
    serverData := make(map[string]interface{})

    content, _, diags := serverBlock.Body.PartialContent(&hcl.BodySchema{
        Attributes: []hcl.AttributeSchema{
            {Name: "data_dir"},
            {Name: "log_level"},
            {Name: "listen"},
            {Name: "terraform_driver"},
        },
        Blocks: []hcl.BlockHeaderSchema{
            {Type: "consul"},
        },
    })
    if diags.HasErrors() {
        fmt.Printf("Failed to get server content: %s\n", diags)
        return nil
    }

    for k, attr := range content.Attributes {
        val, diags := attr.Expr.Value(nil)
        if diags.HasErrors() {
            fmt.Printf("Failed to decode attribute value for %s: %s\n", k, diags)
            continue
        }
        serverData[k] = val.AsString()
    }

    for _, block := range content.Blocks {
        if block.Type == "consul" {
            consulData := processConsulBlock(block)
            if consulData != nil {
                serverData["consul"] = consulData
            }
        }
    }

    return serverData
}

func processConsulBlock(consulBlock *hcl.Block) map[string]interface{} {
    consulData := make(map[string]interface{})

    attrs, diags := consulBlock.Body.JustAttributes()
    if diags.HasErrors() {
        fmt.Printf("Failed to decode consul attributes: %s\n", diags)
        return nil
    }

    for key, attr := range attrs {
        val, diags := attr.Expr.Value(nil)
        if diags.HasErrors() {
            fmt.Printf("Failed to decode attribute value for %s: %s\n", key, diags)
            continue
        }
        consulData[key] = val.AsString()
    }

    return consulData
}

func populateServerStruct(serverMap map[string]interface{}) *Server {
    server := &Server{
        Listen: serverMap["listen"].(string),
        DataDir: serverMap["data_dir"].(string),
        LogLevel: serverMap["log_level"].(string),
        TerraformDriver: serverMap["terraform_driver"].(string),
    }

    if consul, ok := serverMap["consul"].(map[string]interface{}); ok {
        server.Consul = ConsulConfig{
            Address: consul["address"].(string),
        }
    }

    return server
}

func populateTaskStruct(taskMap map[string]interface{}) *Task {
    task := &Task{
        Name:        taskMap["name"].(string),
        Description: taskMap["description"].(string),
        Source:      taskMap["source"].(string),
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

    if labels, ok := condMap["labels"].(map[string]string); ok {
        condition.Labels = labels
    }

    return condition
}

func processTaskBlock(taskBlock *hcl.Block) map[string]interface{} {
    taskData := make(map[string]interface{})

    content, _, diags := taskBlock.Body.PartialContent(&hcl.BodySchema{
        Attributes: []hcl.AttributeSchema{
            {Name: "name"},
            {Name: "description"},
            {Name: "source"},
        },
        Blocks: []hcl.BlockHeaderSchema{
            {Type: "condition", LabelNames: []string{"type"}},
        },
    })
    if diags.HasErrors() {
        fmt.Printf("Failed to get task content: %s\n", diags)
        return nil
    }

    for k, attr := range content.Attributes {
        val, diags := attr.Expr.Value(nil)
        if diags.HasErrors() {
            fmt.Printf("Failed to decode attribute value for %s: %s\n", k, diags)
            continue
        }
        taskData[k] = val.AsString()
    }

    for _, block := range content.Blocks {
        if block.Type == "condition" && len(block.Labels) > 0 && block.Labels[0] == "label-match" {
            conditionData := processConditionBlock(block)
            if conditionData != nil {
                taskData["condition"] = conditionData
            }
        }
    }

    return taskData
}

func processConditionBlock(conditionBlock *hcl.Block) map[string]interface{} {
    conditionData := make(map[string]interface{})

    content, _, diags := conditionBlock.Body.PartialContent(&hcl.BodySchema{
        Attributes: []hcl.AttributeSchema{
            {Name: "notify_on_failure"},
            {Name: "resolved_signal"},
            {Name: "ignore_resolved"},
        },
        Blocks: []hcl.BlockHeaderSchema{{Type: "label"}},
    })
    if diags.HasErrors() {
        fmt.Printf("Failed to get condition content: %s\n", diags)
        return nil
    }

    for k, attr := range content.Attributes {
        val, diags := attr.Expr.Value(nil)
        if diags.HasErrors() {
            fmt.Printf("Failed to decode attribute value for %s: %s\n", k, diags)
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
            labelData := processLabelBlock(labelBlock)
            if labelData != nil {
                conditionData["labels"] = labelData
            }
        }
    }

    return conditionData
}

func processLabelBlock(labelBlock *hcl.Block) map[string]string {
    labels := make(map[string]string)

    attrs, diags := labelBlock.Body.JustAttributes()
    if diags.HasErrors() {
        fmt.Printf("Failed to decode label attributes: %s\n", diags)
        return nil
    }

    for key, attr := range attrs {
        val, diags := attr.Expr.Value(nil)
        if diags.HasErrors() {
            fmt.Printf("Failed to decode attribute value for %s: %s\n", key, diags)
            continue
        }
        labels[key] = val.AsString()
    }

    return labels
}
