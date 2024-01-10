package bootstrap

import (
	"fmt"
  l "log"
	"os"
	"os/signal"

	"github.com/cloudputation/iterator/packages/config"
	log "github.com/cloudputation/iterator/packages/logger"
	"github.com/cloudputation/iterator/packages/server"
	"github.com/cloudputation/iterator/packages/storage"
  "github.com/cloudputation/iterator/packages/storage/consul"
	"github.com/cloudputation/iterator/packages/terraform"
)

const (
    defaultListenAddr = "9595"
    defaultLogLevel   = "info"
)

var GlobalConfig *config.InitConfig

func BootstrapIterator(globalConfigPath string) error {
  GlobalConfig, err := config.LoadConfig(globalConfigPath)
  if err != nil {
      l.Fatal("Error loading config: %w", err)
  }

  // Initialize log
  err = log.InitLogger(GlobalConfig.Server.LogDir, GlobalConfig.Server.LogLevel)
  if err != nil {
      l.Fatal("Error initializing log: %w", err)
  }
  defer log.CloseLogger()

  log.Info("Starting Iterator..")
  log.Info("Log level is: %s", GlobalConfig.Server.LogLevel)
  storage.InitStorage(GlobalConfig)
  terraform.InitTerraform(GlobalConfig)

  ymlConfigPath := fmt.Sprintf("%s/config.yml", GlobalConfig.Server.DataDir)

  // Render YAML configuration based on the loaded HCL config
  err = config.RenderConfig(GlobalConfig, ymlConfigPath)
  if err != nil {
      log.Fatal("Error rendering YAML config: %w", err)
  }
  log.Info("YAML configuration generated successfully")

  // Read YAML config
  c, err := config.ReadConfig(ymlConfigPath)
  if err != nil {
      log.Fatal("Couldn't determine configuration: %w", err)
  }

  if config.ConsulStorageEnabled {
    var consulAddress = GlobalConfig.Server.Consul.Address
    log.Info("Consul storage is enabled!")
    log.Info("Connecting to Consul at address: %s..", consulAddress)
    err = consul.InitConsul(consulAddress)
    if err != nil {
        return fmt.Errorf("Could not initialize Consul: %v", err)
    }

    err = consul.BootstrapConsul()
    if err != nil {
        return fmt.Errorf("Could not bootstrap factory on Consul: %v", err)
    }
  }

  startIterator(GlobalConfig, c)

  return nil
}

func startIterator(initConfig *config.InitConfig, c *config.Config) {
  s := server.NewServer(initConfig, c)

  // Listen for signals telling us to stop
  signals := make(chan os.Signal, 1)
  signal.Notify(signals, os.Interrupt)

  // Start the HTTP server
  srv, srvResult := s.Start()

  select {
  case err := <-srvResult:
    if err != nil {
      log.Fatal("Failed to serve for %s: %w", c.ListenAddr, err)
    } else {
      log.Info("HTTP server shut down")
    }
  case sig := <-signals:
    log.Info("Shutting down due to signal: %s", sig)
    if err := server.StopServer(srv); err != nil {
      log.Info("Failed to shut down HTTP server: %w", err)
    }
  }
}
