package set

import (
	"github.com/spf13/cobra"

	"github.com/weaveworks/weave-gitops/cmd/gitops/config"
	configCmd "github.com/weaveworks/weave-gitops/cmd/gitops/set/config"
)

func SetCommand(opts *config.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Sets one or many Weave GitOps CLI configs or resources",
		Example: `
# Enables analytics in the current user's CLI configuration for Weave GitOps
gitops set config analytics true`,
	}

	cmd.AddCommand(configCmd.ConfigCommand(opts))

	return cmd
}
