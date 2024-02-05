package storage

import (
  "fmt"
  "os"

  "github.com/cloudputation/iterator/packages/config"
  log "github.com/cloudputation/iterator/packages/logger"
)

func InitStorage(cfg *config.InitConfig) {
  dataDir := cfg.Server.DataDir
  executorDir := fmt.Sprintf("%s/process/alerts", dataDir)

  dataDirectories := []string{dataDir, executorDir}

  for _, dir := range dataDirectories {
    if err := createDirectory(dir); err != nil {
      log.Info("Error creating directory %s: %v", dataDir, err)
    }
  }
}

func createDirectory(path string) error {
  err := os.MkdirAll(path, 0755)
  if err != nil {
    return err
  }
  log.Info("Directory created: %s", path)
  return nil
}
