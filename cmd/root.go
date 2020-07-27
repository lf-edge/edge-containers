package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{Use: "eci"}
	reg     string
)

func init() {
	rootCmd.AddCommand(pushCmd)
	pushInit()
	rootCmd.AddCommand(pullCmd)
	pullInit()

	rootCmd.PersistentFlags().StringVar(&reg, "registry", "", "path to registry to use, leave blank to use remote")

}

// Execute primary function for cobra
func Execute() {
	rootCmd.Execute()
}
