package main

import (
	"fmt"
	"os"

	"generateK8sPatchfile/cmd"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "generateK8sPatchfile",
		Short: "CLI tool to generate K8s patch filesfor resource modification",
		Long:  "A command-line tool to interactively generate YAML configuration files for Velero resource modifiers.",
	}

	// Add commands
	rootCmd.AddCommand(cmd.GenerateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
