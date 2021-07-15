package app

import (
	"github.com/spf13/cobra"
	"github.com/weaveworks/weave-gitops/cmd/wego/app/add"
	"github.com/weaveworks/weave-gitops/cmd/wego/app/list"
	"github.com/weaveworks/weave-gitops/cmd/wego/app/status"
)

var ApplicationCmd = &cobra.Command{
	Use:   "app",
	Short: "Manages your applications",
	Example:`
  # Add an application to wego from local git repository
  wego app add . --name <app-name>

  # Status an application under wego control
  wego app status <app-name>

  # List applications under wego control
  wego app list`,
	Args: cobra.MinimumNArgs(1),
}

func init() {
	ApplicationCmd.AddCommand(status.Cmd)
	ApplicationCmd.AddCommand(add.Cmd)
	ApplicationCmd.AddCommand(list.Cmd)
}
