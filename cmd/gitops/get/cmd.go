package get

import (
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/weaveworks/weave-gitops/cmd/gitops/get/templates"
)

func GetCommand(endpoint string, client *resty.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many Weave GitOps resources",
		Example: `
# Get all CAPI templates that are available on the cluster
gitops get templates`,
	}

	cmd.AddCommand(templates.TemplateCommand(endpoint, client))

	return cmd
}
