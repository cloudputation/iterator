package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/cloudputation/iterator/packages/storage"
	"github.com/cloudputation/iterator/packages/terraform"
	"github.com/cloudputation/iterator/packages/config"
	"github.com/cloudputation/iterator/packages/server"
)

var GlobalConfig *config.InitConfig

const (
	// How long we are willing to wait for the HTTP server to shut down gracefully
	serverShutdownTime = time.Second * 4
)

var configFile = "./config.yml"

func init() {
  var err error
  GlobalConfig, err = config.LoadConfig(".release/defaults/config.hcl")
  if err != nil {
      log.Fatalf("Error loading config: %v", err)
  }
	storage.InitStorage(GlobalConfig)
	terraform.InitTerraform(GlobalConfig)
}

// stopServer issues a time-limited server shutdown
func stopServer(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTime)
	defer cancel()
	return srv.Shutdown(ctx)
}

func main() {
	// Render YAML configuration based on the loaded HCL config
	err := config.RenderConfig(GlobalConfig)
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
		if err := stopServer(srv); err != nil {
			log.Printf("Failed to shut down HTTP server: %v", err)
		}
	}
}
