package terraform

import (
    "bytes"
    "fmt"
    l "log"
    "os/exec"

    "github.com/cloudputation/iterator/packages/config"
    log "github.com/cloudputation/iterator/packages/logger"
)

var terraformRoutine = []string{"init", "plan"}

func InitTerraform(cfg *config.InitConfig) {
  log.Info("Initializing Terraform..")
  for _, task := range cfg.Tasks {
      go func(t *config.Task) {
          moduleDir := t.Source
          for _, command := range terraformRoutine {
              if err := runTerraformInitRoutine(moduleDir, command); err != nil {
                  log.Error("Failed to initialize Terraform module %s: %v", moduleDir, err)
              }
          }
      }(task)
  }
}

func runTerraformInitRoutine(moduleDir, terraformCommand string) error {
  terraformModulePath := fmt.Sprintf("-chdir=%s", moduleDir)
  cmd := exec.Command("terraform", terraformModulePath, terraformCommand)

  var stdout, stderr bytes.Buffer
  cmd.Stdout = &stdout
  cmd.Stderr = &stderr

  err := cmd.Run()

  l.Printf("Terraform stdout: %s", stdout.String())
  if stderr.String() != "" {
    log.Error("Terraform stderr: %s", stderr.String())
    log.Fatal("Terraform stderr: %s", stderr.String())
  }

  if err != nil {
      return fmt.Errorf("Failed to run Terraform %s on module: %s: %v", terraformCommand, moduleDir, err)
  }

  if terraformCommand =="init" {
    l.Printf("Initialized Terraform directory for module: %s", moduleDir)
  }
  if terraformCommand =="plan" {
    log.Info("Ran Terraform plan for module: %s", moduleDir)
  }

  return nil
}

func RunTerraform(moduleDir, terraformCommand string) error {
  terraformModulePath := fmt.Sprintf("-chdir=%s", moduleDir)
  cmd := exec.Command("terraform", terraformModulePath, terraformCommand, "-auto-approve")

  var stdout, stderr bytes.Buffer
  cmd.Stdout = &stdout
  cmd.Stderr = &stderr

  err := cmd.Run()

  l.Printf("Terraform stdout: %s", stdout.String())
  if stderr.String() != "" {
    log.Error("Terraform stderr: %s", stderr.String())
    log.Fatal("Terraform stderr: %s", stderr.String())
  }

  if err != nil {
      return fmt.Errorf("Failed to run Terraform %s on module: %s: %v", terraformCommand, moduleDir, err)
  }

  log.Info("Executed Terraform %s on module: %s", terraformCommand, moduleDir)

  return nil
}
