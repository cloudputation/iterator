package storage

import (
    "fmt"
    "os"

    "github.com/cloudputation/iterator/packages/config"
)

func InitStorage(cfg *config.InitConfig) {
  dataDir := cfg.Server.DataDir
  executorDir := fmt.Sprintf("%s/executor/map/fingerprints", dataDir)

  dataDirectories := []string{dataDir, executorDir}

  for _, dir := range dataDirectories {
      if err := createDirectory(dir); err != nil {
          fmt.Printf("Error creating directory %s: %v\n", dataDir, err)
      }
  }
}

func createDirectory(path string) error {
  err := os.MkdirAll(path, 0755)
  if err != nil {
      return err
  }
  fmt.Printf("Directory created: %s\n", path)
  return nil
}
