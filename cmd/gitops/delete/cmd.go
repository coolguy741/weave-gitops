package delete

import (
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/weaveworks/weave-gitops/cmd/gitops/delete/app"
	"github.com/weaveworks/weave-gitops/cmd/gitops/delete/clusters"
)

func DeleteCommand(endpoint *string, client *resty.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete one or many Weave GitOps resources",
		Example: `
# Delete an application from gitops
gitops delete app <app-name>

# Delete a CAPI cluster given its name
gitops delete cluster <cluster-name>`,
	}

	cmd.AddCommand(clusters.ClusterCommand(endpoint, client))
	cmd.AddCommand(app.Cmd)

	return cmd
}
