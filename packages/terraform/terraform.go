package terraform

import (
    "bytes"
    "fmt"
    l "log"
    "os/exec"

    "github.com/cloudputation/iterator/packages/config"
    log "github.com/cloudputation/iterator/packages/logger"
)

var terraformInitRoutine = []string{"init", "plan"}
var terraformDirArg string

func InitTerraform(cfg *config.InitConfig) {
  log.Info("Initializing Terraform..")
  terraformDriver := cfg.Server.TerraformDriver
  for _, task := range cfg.Tasks {
      go func(t *config.Task) {
          moduleDir := t.Source
          for _, command := range terraformInitRoutine {
              if err := RunTerraform(terraformDriver, moduleDir, command); err != nil {
                  log.Error("Failed to initialize Terraform module %s: %v", moduleDir, err)
              }
          }
      }(task)
  }
}

func RunTerraform(terraformDriver, moduleDir, terraformCommand string) error {
  switch {
  case terraformDriver == "terraform":
    terraformDirArg = "-chdir="
  case terraformDriver == "terragrunt":
    terraformDirArg = "-config-dir "
  }
  terraformModulePath := fmt.Sprintf("%s%s", terraformDirArg, moduleDir)
  cmdArgs := []string{terraformModulePath, terraformCommand}

  if terraformCommand == "apply" || terraformCommand == "destroy" {
    cmdArgs = append(cmdArgs, "-auto-approve")
  }

  cmd := exec.Command(terraformDriver, cmdArgs...)

  var stdout, stderr bytes.Buffer
  cmd.Stdout = &stdout
  cmd.Stderr = &stderr

  err := cmd.Run()

  // Do not use logger for Terraform stdout to prevent clogging the log file
  l.Printf("Terraform stdout: %s", stdout.String())
  if stderr.String() != "" {
    log.Error("Terraform stderr: %s", stderr.String())
  }

  if err != nil {
    return fmt.Errorf("Failed to run Terraform %s on module: %s: %v", terraformCommand, moduleDir, err)
  }

  if terraformCommand == "init" {
    l.Printf("Initialized Terraform directory for module: %s", moduleDir)
  }

  if terraformCommand == "plan" {
    log.Info("Ran Terraform plan for module: %s", moduleDir)
  }

  if terraformCommand == "apply" || terraformCommand == "destroy" {
    log.Info("Executed Terraform %s on module: %s", terraformCommand, moduleDir)
  }

  return nil
}
