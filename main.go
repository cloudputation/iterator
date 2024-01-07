package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/cloudputation/iterator/packages/config"
	log "github.com/cloudputation/iterator/packages/logger"
	"github.com/cloudputation/iterator/packages/server"
	"github.com/cloudputation/iterator/packages/storage"
	"github.com/cloudputation/iterator/packages/terraform"
)

var GlobalConfig *config.InitConfig
var logLevel = "info"

func init() {
  var err error
  GlobalConfig, err = config.LoadConfig(".release/defaults/test.config.hcl")
  if err != nil {
      log.Fatal("Error loading config: %v", err)
  }
	if GlobalConfig.Server.LogLevel != " " {
		logLevel =  GlobalConfig.Server.LogLevel
	}

	logDirPath := GlobalConfig.Server.LogDir
	log.InitLogger(logDirPath, logLevel)
	log.Info("Starting Iterator..")
	log.Info("Log level is: %s", logLevel)
	storage.InitStorage(GlobalConfig)
	terraform.InitTerraform(GlobalConfig)
}

func main() {
	var configFile = fmt.Sprintf("%s/config.yml", GlobalConfig.Server.DataDir)
	// Render YAML configuration based on the loaded HCL config
	err := config.RenderConfig(GlobalConfig, configFile)
	if err != nil {
			log.Fatal("Error rendering YAML config: %v", err)
	}
	log.Info("YAML configuration generated successfully")

	// Read YAML config
	c, err := config.ReadConfig(configFile)
	if err != nil {
			log.Fatal("Couldn't determine configuration: %v", err)
	}
	s := server.NewServer(GlobalConfig, c)

	// Listen for signals telling us to stop
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// Start the HTTP server
	srv, srvResult := s.Start()

	select {
	case err := <-srvResult:
		if err != nil {
			log.Fatal("Failed to serve for %s: %v", c.ListenAddr, err)
		} else {
			log.Info("HTTP server shut down")
		}
	case sig := <-signals:
		log.Info("Shutting down due to signal: %s", sig)
		if err := server.StopServer(srv); err != nil {
			log.Info("Failed to shut down HTTP server: %v", err)
		}
	}
}
