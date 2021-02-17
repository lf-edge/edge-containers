package cmd

import (
	"context"
	"log"
	"strings"

	ecresolver "github.com/lf-edge/edge-containers/pkg/resolver"
	"github.com/spf13/cobra"
)

var (
	remote       string
	remoteTarget ecresolver.ResolverCloser
	ctrNamespace string
)

var rootCmd = &cobra.Command{
	Use:   "eci",
	Short: "utility to manage an ECI, either push to or pull from a registry",
	Long: `Utility to manage an ECI, either push to or pull from a registry. The registry can be the default based on
	its tag, e.g. library/alpine:3.11 would go to docker hub, a local directory cache, or containerd. Use the --remote
	flag to indicate where to go. Blank ("") is the default registry, /path or file:///path is for a local directory,
	containerd:/path/to/socket is for containerd.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var (
			err error
			ctx = context.TODO()
		)
		switch {
		case remote == "":
			_, remoteTarget, err = ecresolver.NewRegistry(ctx)
		case strings.HasPrefix(remote, "containerd:"):
			_, remoteTarget, err = ecresolver.NewContainerd(ctx, strings.Replace(remote, "containerd:", "", 1), ctrNamespace)
		case strings.HasPrefix(remote, "file://"):
			_, remoteTarget, err = ecresolver.NewDirectory(ctx, strings.Replace(remote, "file://", "", 1))
		case strings.HasPrefix(remote, "/"):
			_, remoteTarget, err = ecresolver.NewDirectory(ctx, remote)
		default:
			log.Fatalf("unknown remote: %s", remote)
		}
		if err != nil {
			log.Fatalf("unexpected error when created NewRegistry resolver: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushInit()
	rootCmd.AddCommand(pullCmd)
	pullInit()
	rootCmd.AddCommand(pullFilesCmd)
	pullFilesInit()

	rootCmd.PersistentFlags().StringVar(&remote, "remote", "", "remote to use for push/pull, leave blank to use default registry for image")
	rootCmd.PersistentFlags().StringVar(&ctrNamespace, "namespace", "default", "namespace to use for containerd, ignored for all other remotes")

}

// Execute primary function for cobra
func Execute() {
	rootCmd.Execute()
}
