package main

import (
		l "log"
    "github.com/cloudputation/iterator/packages/cli"
)

func main() {
    rootCmd := cli.SetupRootCommand()
    if err := rootCmd.Execute(); err != nil {
        l.Fatal("Error executing command: %v", err)
    }
}
