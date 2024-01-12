package cli

import (
    "github.com/spf13/cobra"

    "github.com/cloudputation/iterator/packages/bootstrap"
    log "github.com/cloudputation/iterator/packages/logger"
)

func SetupRootCommand() *cobra.Command {
  var configFile string

  var rootCmd = &cobra.Command{
      Use:   "iterator",
      Short: "Run Terraform using alerts",
      Run: func(cmd *cobra.Command, args []string) {
          if configFile == "" {
              configFile = "/etc/iterator/config.hcl"
          }
					err := bootstrap.BootstrapIterator(configFile)
          if err != nil {
              log.Error("Failed to bootstrap iterator: %v", err)
          }
      },
  }
  rootCmd.Flags().StringVarP(&configFile, "config", "f", "", "Path to configuration file")
  rootCmd.CompletionOptions.HiddenDefaultCmd = true

  return rootCmd
}
