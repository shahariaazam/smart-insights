// main.go
package main

import (
	"log"

	"github.com/shahariaazam/smart-insights/cmd"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "smart-insights",
		Version: formatVersion(),
	}

	rootCmd.AddCommand(cmd.NewStartCommand(version))

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("could not run the application: %v", err)
	}
}

func formatVersion() string {
	return version + " (commit: " + commit + ", built at: " + date + ")"
}
