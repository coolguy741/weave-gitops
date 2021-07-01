package list

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/runner"
)

var Cmd = &cobra.Command{
	Use:     "list",
	Short:   "List applications",
	Example: "wego app list",
	RunE:    runCmd,
}

func runCmd(cmd *cobra.Command, args []string) error {

	kubeClient := kube.New(&runner.CLIRunner{})

	ns, err := cmd.Parent().Parent().Flags().GetString("namespace")
	if err != nil {
		return err
	}

	apps, err := kubeClient.GetApplications(ns)
	if err != nil {
		return err
	}

	fmt.Println("NAME")
	for _, app := range *apps {
		fmt.Println(app.Name)
	}

	return nil
}
