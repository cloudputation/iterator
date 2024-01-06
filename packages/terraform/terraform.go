package terraform

import (
    "bytes"
    "fmt"
    "log"
    "os/exec"

    "github.com/cloudputation/iterator/packages/config"
)

var terraformRoutine = []string{"init", "plan"}

func InitTerraform(cfg *config.InitConfig) {
  for _, task := range cfg.Tasks {
      go func(t *config.Task) {
          moduleDir := t.Source
          for _, command := range terraformRoutine {
              log.Printf("Initiating Terraform directory for module: %s\n", moduleDir)
              if err := runTerraformInitRoutine(moduleDir, command); err != nil {
                  log.Printf("Error initiating Terraform module %s: %v", moduleDir, err)
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

  log.Printf("Initializing Terraform directory for module: %s\n", moduleDir)
  err := cmd.Run()

  log.Printf("Terraform stdout: %s", stdout.String())
  log.Printf("Terraform stderr: %s", stderr.String())

  if err != nil {
      return fmt.Errorf("error executing Terraform command: %s: %v", terraformCommand, err)
  }

  return nil
}

func RunTerraform(moduleDir, terraformCommand string) error {
  terraformModulePath := fmt.Sprintf("-chdir=%s", moduleDir)
  cmd := exec.Command("terraform", terraformModulePath, terraformCommand, "-auto-approve")

  var stdout, stderr bytes.Buffer
  cmd.Stdout = &stdout
  cmd.Stderr = &stderr

  log.Printf("Initializing Terraform directory for module: %s\n", moduleDir)
  err := cmd.Run()

  log.Printf("Terraform stdout: %s", stdout.String())
  log.Printf("Terraform stderr: %s", stderr.String())

  if err != nil {
      return fmt.Errorf("error executing Terraform command: %s: %v", terraformCommand, err)
  }

  return nil
}
