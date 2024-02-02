package main

import (
    "os"
    "github.com/cloudputation/iterator/packages/cli"
)

func main() {
    app := cli.NewApp()

    // Execute the root command
    if err := app.RootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
