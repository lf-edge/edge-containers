package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{Use: "eci"}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushInit()
	rootCmd.AddCommand(pullCmd)
	pullInit()
}

// Execute primary function for cobra
func Execute() {
	rootCmd.Execute()
}
