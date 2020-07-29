package cmd

import (
	"log"
	"strings"

	"github.com/lf-edge/edge-containers/pkg/registry/target"
	"github.com/spf13/cobra"
)

var (
	remote       string
	remoteTarget target.Target
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
		switch {
		case remote == "":
			remoteTarget = target.Registry{}
		case strings.HasPrefix(remote, "containerd:"):
			remoteTarget = target.NewContainerd(strings.Replace(remote, "containerd:", "", 1), ctrNamespace)
		case strings.HasPrefix(remote, "file://"):
			remoteTarget = target.NewDirectory(strings.Replace(remote, "file://", "", 1))
		case strings.HasPrefix(remote, "/"):
			remoteTarget = target.NewDirectory(remote)
		default:
			log.Fatalf("unknown remote: %s", remote)
		}
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushInit()
	rootCmd.AddCommand(pullCmd)
	pullInit()

	rootCmd.PersistentFlags().StringVar(&remote, "remote", "", "remote to use for push/pull, leave blank to use default registry for image")
	rootCmd.PersistentFlags().StringVar(&ctrNamespace, "namespace", "default", "namespace to use for containerd, ignored for all other remotes")

}

// Execute primary function for cobra
func Execute() {
	rootCmd.Execute()
}
