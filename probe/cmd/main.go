package main

import (
	"log"

	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	"github.com/spf13/cobra"
	_ "go.uber.org/automaxprocs"
)

var rootCmd = &cobra.Command{
	Use:   "buildkit-state",
	Short: "Manage buildkit state intrusively",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("%+v\n", err)
	}
}
