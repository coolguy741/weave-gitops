package add

// Provides support for adding an application to wego managment.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lithammer/dedent"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/weaveworks/weave-gitops/cmd/wego/version"
	"github.com/weaveworks/weave-gitops/pkg/flux"
	"github.com/weaveworks/weave-gitops/pkg/git"
	"github.com/weaveworks/weave-gitops/pkg/git/wrapper"
	"github.com/weaveworks/weave-gitops/pkg/gitproviders"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/logger"
	"github.com/weaveworks/weave-gitops/pkg/osys"
	"github.com/weaveworks/weave-gitops/pkg/runner"
	"github.com/weaveworks/weave-gitops/pkg/services/app"
	"github.com/weaveworks/weave-gitops/pkg/services/auth"
	"github.com/weaveworks/weave-gitops/pkg/utils"
)

const (
	SSHAuthSock = "SSH_AUTH_SOCK"
)

var params app.AddParams

var Cmd = &cobra.Command{
	Use:   "add [--name <name>] [--url <url>] [--branch <branch>] [--path <path within repository>] [--private-key <keyfile>] <repository directory>",
	Short: "Add a workload repository to a wego cluster",
	Long: strings.TrimSpace(dedent.Dedent(`
        Associates an additional application in a git repository with a wego cluster so that its contents may be managed via GitOps
    `)),
	Example: `
  # Add application to wego control from local git repository
  wego app add .

  # Add podinfo application to wego control from github repository
  wego app add --url git@github.com:myorg/podinfo

  # Get status of podinfo application
  wego app status podinfo
`,
	RunE:          runCmd,
	SilenceUsage:  true,
	SilenceErrors: true,
	PostRun: func(cmd *cobra.Command, args []string) {
		version.CheckVersion(version.CheckpointParamsWithFlags(version.CheckpointParams(), cmd))
	},
}

func init() {
	Cmd.Flags().StringVar(&params.Name, "name", "", "Name of application")
	Cmd.Flags().StringVar(&params.Url, "url", "", "URL of remote repository")
	Cmd.Flags().StringVar(&params.Path, "path", app.DefaultPath, "Path of files within git repository")
	Cmd.Flags().StringVar(&params.Branch, "branch", app.DefaultBranch, "Branch to watch within git repository")
	Cmd.Flags().StringVar(&params.DeploymentType, "deployment-type", app.DefaultDeploymentType, "deployment type [kustomize, helm]")
	Cmd.Flags().StringVar(&params.Chart, "chart", "", "Specify chart for helm source")
	Cmd.Flags().StringVar(&params.PrivateKey, "private-key", "", "Private key to access git repository over ssh")
	Cmd.Flags().StringVar(&params.AppConfigUrl, "app-config-url", "", "URL of external repository (if any) which will hold automation manifests; NONE to store only in the cluster")
	Cmd.Flags().StringVar(&params.HelmReleaseTargetNamespace, "helm-release-target-namespace", "", "Namespace in which to deploy a helm chart; defaults to the wego installation namespace")
	Cmd.Flags().BoolVar(&params.DryRun, "dry-run", false, "If set, 'wego add' will not make any changes to the system; it will just display the actions that would have been taken")
	Cmd.Flags().BoolVar(&params.AutoMerge, "auto-merge", false, "If set, 'wego add' will merge automatically into the set --branch")
}

func runCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	params.Namespace, _ = cmd.Parent().Flags().GetString("namespace")

	if params.Url != "" && len(args) > 0 {
		return fmt.Errorf("you should choose either --url or the app directory")
	}

	if len(args) > 0 {
		path, err := filepath.Abs(args[0])
		if err != nil {
			return errors.Wrap(err, "failed to get absolute path for the repo directory")
		}

		params.Dir = path
	}

	var err error
	osysClient := osys.New()
	cliRunner := &runner.CLIRunner{}
	fluxClient := flux.New(osysClient, cliRunner)
	kubeClient := kube.New(cliRunner)
	kube, rawClient, err := kube.NewKubeHTTPClient()
	if err != nil {
		return fmt.Errorf("error creating k8s http client: %w", err)
	}

	logger := logger.NewCLILogger(os.Stdout)
	if err := app.IsClusterReady(logger, kube); err != nil {
		return err
	}

	isHelmRepository := params.Chart != ""
	repoUrlString := params.Url
	if repoUrlString == "" {
		// Find the url using an unauthenticated git client. We just need to read the URL.
		// params.Dir must be defined here because we already checked for it above.
		repoUrlString, err = git.New(nil, wrapper.NewGoGit()).GetRemoteUrl(params.Dir, "origin")
		if err != nil {
			return fmt.Errorf("could not get remote url for directory %s: %w", params.Dir, err)
		}
	}

	var providerName gitproviders.GitProviderName
	// We re-use the same --url flag for both git and helm sources.
	// There isn't really a concept of "provider" in helm charts, and there is nothing to push.
	// Assume charts are always public and no auth needs to be done.
	if !isHelmRepository {
		providerName, err = gitproviders.DetectGitProviderFromUrl(repoUrlString)
		if err != nil {
			return fmt.Errorf("error detecting git provider: %w", err)
		}
	}

	token, tokenErr := osysClient.GetGitProviderToken()

	if !isHelmRepository && tokenErr == osys.ErrNoGitProviderTokenSet {
		// No provider token set, we need to do the auth flow.
		authHandler, err := auth.NewAuthCLIHandler(providerName)
		if err != nil {
			return fmt.Errorf("could not get auth handler for provider %s: %w", providerName, err)
		}

		token, err = authHandler(ctx, osysClient.Stdout())
		if err != nil {
			return fmt.Errorf("could not complete auth flow: %w", err)
		}
	} else if !isHelmRepository && tokenErr != nil {
		// We didn't detect a NoGitProviderSet error, something else went wrong.
		return fmt.Errorf("could not get access token: %w", err)
	}

	params.GitProviderToken = token

	authMethod, err := osysClient.SelectAuthMethod(params.PrivateKey)
	if err != nil {
		return fmt.Errorf("error selecting auth method: %w", err)
	}

	gitClient := git.New(authMethod, wrapper.NewGoGit())

	// If we are NOT doing a helm chart, we want to use a git client with an embedded deploy key
	if !isHelmRepository {
		authsvc, err := auth.NewAuthService(fluxClient, rawClient, providerName, logger, token)
		if err != nil {
			return fmt.Errorf("error creating auth service: %w", err)
		}

		targetName, err := kubeClient.GetClusterName(ctx)
		if err != nil {
			return fmt.Errorf("error getting target name: %w", err)
		}

		normalizedUrl, err := gitproviders.NewNormalizedRepoURL(repoUrlString)
		if err != nil {
			return fmt.Errorf("error creating normalized url: %w", err)
		}

		name := auth.SecretName{
			Name:      app.CreateAppSecretName(targetName, normalizedUrl.String()),
			Namespace: params.Namespace,
		}
		gitClient, err = authsvc.SetupDeployKey(ctx, name, targetName, normalizedUrl)
		if err != nil {
			return fmt.Errorf("error setting up deploy keys: %w", err)
		}

	}

	appService := app.New(logger, gitClient, fluxClient, kubeClient, osysClient)

	utils.SetCommmitMessageFromArgs("wego app add", params.Url, params.Path, params.Name)

	if err := appService.Add(params); err != nil {
		return errors.Wrapf(err, "failed to add the app %s", params.Name)
	}

	return nil
}
