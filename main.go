package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/cloudputation/iterator/packages/storage"
	"github.com/cloudputation/iterator/packages/terraform"
	"github.com/cloudputation/iterator/packages/config"
	"github.com/cloudputation/iterator/packages/server"
)

var GlobalConfig *config.InitConfig

func init() {
  var err error
  GlobalConfig, err = config.LoadConfig(".release/defaults/test.config.hcl")
  if err != nil {
      log.Fatalf("Error loading config: %v", err)
  }
	storage.InitStorage(GlobalConfig)
	terraform.InitTerraform(GlobalConfig)
}

func main() {
	var configFile = fmt.Sprintf("%s/config.yml", GlobalConfig.Server.DataDir)
	// Render YAML configuration based on the loaded HCL config
	err := config.RenderConfig(GlobalConfig, configFile)
	if err != nil {
			log.Fatalf("Error rendering YAML config: %v", err)
	}
	log.Println("YAML configuration generated successfully")

	// Read YAML config
	c, err := config.ReadConfig(configFile)
	if err != nil {
			log.Fatalf("Couldn't determine configuration: %v", err)
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
			log.Fatalf("Failed to serve for %s: %v", c.ListenAddr, err)
		} else {
			log.Println("HTTP server shut down")
		}
	case sig := <-signals:
		log.Println("Shutting down due to signal:", sig)
		if err := server.StopServer(srv); err != nil {
			log.Printf("Failed to shut down HTTP server: %v", err)
		}
	}
}
