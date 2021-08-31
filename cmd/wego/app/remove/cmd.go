package remove

// Provides support for removing an application from wego management.

import (
	"fmt"
	"os"
	"strings"

	"github.com/weaveworks/weave-gitops/pkg/git/wrapper"

	"github.com/lithammer/dedent"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/weaveworks/weave-gitops/cmd/wego/version"
	"github.com/weaveworks/weave-gitops/pkg/flux"
	"github.com/weaveworks/weave-gitops/pkg/git"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/logger"
	"github.com/weaveworks/weave-gitops/pkg/osys"
	"github.com/weaveworks/weave-gitops/pkg/runner"
	"github.com/weaveworks/weave-gitops/pkg/services/app"
	"github.com/weaveworks/weave-gitops/pkg/utils"
)

var params app.RemoveParams

var Cmd = &cobra.Command{
	Use:   "remove [--private-key <keyfile>] <app name>",
	Short: "Remove an app from a wego cluster",
	Long: strings.TrimSpace(dedent.Dedent(`
        Removes an application from a wego cluster so it will no longer be managed via GitOps
    `)),
	Example: `
  # Remove application from wego control via immediate commit
  wego app remove podinfo
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
	Cmd.Flags().StringVar(&params.Name, "name", "", "Name of application")
	Cmd.Flags().StringVar(&params.PrivateKey, "private-key", "", "Private key to access git repository over ssh")
	Cmd.Flags().BoolVar(&params.DryRun, "dry-run", false, "If set, 'wego remove' will not make any changes to the system; it will just display the actions that would have been taken")
}

func runCmd(cmd *cobra.Command, args []string) error {
	params.Name = args[0]
	params.Namespace, _ = cmd.Parent().Flags().GetString("namespace")

	osysClient := osys.New()

	token, err := osysClient.GetGitProviderToken()
	if err != nil {
		return err
	}

	params.GitProviderToken = token

	authMethod, err := osysClient.SelectAuthMethod(params.PrivateKey)
	if err != nil {
		return err
	}

	cliRunner := &runner.CLIRunner{}
	fluxClient := flux.New(osysClient, cliRunner)
	kubeClient, _, err := kube.NewKubeHTTPClient()
	if err != nil {
		return fmt.Errorf("error initializing kube client: %w", err)
	}
	gitClient := git.New(authMethod, wrapper.NewGoGit())
	logger := logger.NewCLILogger(os.Stdout)

	appService := app.New(logger, gitClient, fluxClient, kubeClient, osysClient)

	utils.SetCommmitMessage(fmt.Sprintf("wego app remove %s", params.Name))

	if err := appService.Remove(params); err != nil {
		return errors.Wrapf(err, "failed to remove the app %s", params.Name)
	}

	return nil
}
