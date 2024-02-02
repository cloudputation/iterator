package bootstrap

import (
    "fmt"
    "os"
    "os/signal"

    "github.com/cloudputation/iterator/packages/config"
    "github.com/cloudputation/iterator/packages/consul"
    log "github.com/cloudputation/iterator/packages/logger"
    "github.com/cloudputation/iterator/packages/server"
    "github.com/cloudputation/iterator/packages/stats"
    "github.com/cloudputation/iterator/packages/storage"
    "github.com/cloudputation/iterator/packages/terraform"
)

func BootstrapIterator(initConfig *config.InitConfig) error {
    log.Info("Starting Iterator..")
    log.Info("Log level is: %s", initConfig.Server.LogLevel)

    storage.InitStorage(initConfig)
    terraform.InitTerraform(initConfig)

    ymlConfigPath := fmt.Sprintf("%s/config.yml", initConfig.Server.DataDir)

    // Render YAML configuration based on the loaded HCL config
    err := config.RenderConfig(initConfig, ymlConfigPath)
    if err != nil {
        log.Fatal("Error rendering YAML config: %w", err)
    }
    log.Info("YAML configuration generated successfully")

    // Read YAML config
    c, err := config.ReadConfig(ymlConfigPath)
    if err != nil {
        log.Fatal("Couldn't determine configuration: %w", err)
    }

    if initConfig.Server.Consul.Address != "" {
        log.Info("Consul storage is enabled!")
        log.Info("Connecting to Consul at address: %s..", initConfig.Server.Consul.Address)
        err = consul.InitConsul(initConfig.Server.Consul.Address)
        if err != nil {
            return fmt.Errorf("Could not initialize Consul: %v", err)
        }

        err = consul.BootstrapConsul()
        if err != nil {
            return fmt.Errorf("Could not bootstrap factory on Consul: %v", err)
        }
    }

    err = stats.UpdateStatusWithActiveAlerts()
    if err != nil {
        return fmt.Errorf("Failed to update Iterator with alerts status: %v", err)
    }

    startIterator(initConfig, c)

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
