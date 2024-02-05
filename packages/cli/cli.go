package cli

import (
  "fmt"
  l "log"
  "github.com/spf13/cobra"

  "github.com/cloudputation/iterator/packages/bootstrap"
  "github.com/cloudputation/iterator/packages/config"
  log "github.com/cloudputation/iterator/packages/logger"
)

type App struct {
  RootCmd *cobra.Command
  Config *config.InitConfig
}

func NewApp() *App {
  app := &App{}
  var configFile string
  var err error

  app.RootCmd = &cobra.Command{
    Use:   "iterator",
    Short: "Run Terraform using alerts",
    PersistentPreRun: func(cmd *cobra.Command, args []string) {
      configFile, _ := cmd.Flags().GetString("config")
      if configFile == "" {
        configFile = "/etc/iterator/config.hcl"
      }

      // Load configuration file
      app.Config, err = config.LoadConfig(configFile)
      if err != nil {
        l.Fatal("Failed to load config: %v", err)
      }

      // Initialize logger
      err = log.InitLogger(app.Config.Server.LogDir, app.Config.Server.LogLevel)
      if err != nil {
        l.Fatal("Failed to initialize logger: %v", err)
      }
    },
    Run: func(cmd *cobra.Command, args []string) {
      err = bootstrap.BootstrapIterator(app.Config)
      if err != nil {
        log.Error("Failed to bootstrap iterator: %v", err)
      }
    },
  }

  // Define a flag for the config file
  app.RootCmd.PersistentFlags().StringVarP(&configFile, "config", "f", "", "Path to configuration file")
  app.RootCmd.CompletionOptions.HiddenDefaultCmd = true

  app.setupCommands()
  return app
}

func (app *App) setupCommands() {
  var releaseCmd = &cobra.Command{
    Use:   "release [alert name]",
    Short: "Release Terraform resources for the specified alert",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
      alertName := args[0]
      if err := app.handleRelease(alertName); err != nil {
        fmt.Printf("Failed to handle release: %v\n", err)
      }
    },
  }

  app.RootCmd.AddCommand(releaseCmd)
}
