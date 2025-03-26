package main

import (
	"fmt"
	"os"

	"yamltool/cmd"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "yamltool",
		Short: "CLI tool to generate Velero resource modifier YAML files",
		Long:  "A command-line tool to interactively generate YAML configuration files for Velero resource modifiers.",
	}

	// Add commands
	rootCmd.AddCommand(cmd.GenerateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
