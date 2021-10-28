package app

// Provides support for removing an application from gitops management.

import (
	"context"
	"fmt"
	"strings"

	"github.com/lithammer/dedent"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/weaveworks/weave-gitops/cmd/gitops/version"
	"github.com/weaveworks/weave-gitops/pkg/apputils"
	"github.com/weaveworks/weave-gitops/pkg/services/app"
)

var params app.RemoveParams

var Cmd = &cobra.Command{
	Use:   "app <app name>",
	Short: "Delete an app from a gitops cluster",
	Long: strings.TrimSpace(dedent.Dedent(`
        Deletes an application from a gitops cluster so it will no longer be managed via GitOps
    `)),
	Example: `
  # Delete application from gitops control via immediate commit
  gitops delete app podinfo
`,
	Args:          cobra.MinimumNArgs(1),
	RunE:          runCmd,
	SilenceUsage:  true,
	SilenceErrors: true,
	PostRun: func(cmd *cobra.Command, args []string) {
		version.CheckVersion(version.CheckpointParamsWithFlags(version.CheckpointParams(), cmd))
	},
}

func init() {
	Cmd.Flags().BoolVar(&params.DryRun, "dry-run", false, "If set, 'gitops delete app' will not make any changes to the system; it will just display the actions that would have been taken")
}

func runCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	params.Name = args[0]
	params.Namespace, _ = cmd.Parent().Flags().GetString("namespace")

	appService, appError := apputils.GetAppService(ctx, params.Name, params.Namespace)
	if appError != nil {
		return fmt.Errorf("failed to create app service: %w", appError)
	}

	if err := appService.Remove(params); err != nil {
		return errors.Wrapf(err, "failed to delete the app %s", params.Name)
	}

	return nil
}
