package install

// Provides support for adding a repository of manifests to a gitops cluster. If the cluster does not have
// gitops installed, the user will be prompted to install gitops and then the repository will be added.

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	wego "github.com/weaveworks/weave-gitops/api/v1alpha1"
	"github.com/weaveworks/weave-gitops/cmd/gitops/version"
	"github.com/weaveworks/weave-gitops/cmd/internal"
	"github.com/weaveworks/weave-gitops/pkg/flux"
	"github.com/weaveworks/weave-gitops/pkg/git"
	"github.com/weaveworks/weave-gitops/pkg/git/wrapper"
	"github.com/weaveworks/weave-gitops/pkg/gitproviders"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/models"
	"github.com/weaveworks/weave-gitops/pkg/osys"
	"github.com/weaveworks/weave-gitops/pkg/runner"
	"github.com/weaveworks/weave-gitops/pkg/services/auth"
	"github.com/weaveworks/weave-gitops/pkg/services/gitopswriter"
	"github.com/weaveworks/weave-gitops/pkg/services/install"
	corev1 "k8s.io/api/core/v1"
)

type params struct {
	DryRun     bool
	AutoMerge  bool
	ConfigRepo string
}

var (
	installParams params
)

var Cmd = &cobra.Command{
	Use:   "install",
	Short: "Install or upgrade GitOps",
	Long: `The install command deploys GitOps in the specified namespace,
adds a cluster entry to the GitOps repo, and persists the GitOps runtime into the
repo. If a previous version is installed, then an in-place upgrade will be performed.`,
	Example: fmt.Sprintf(`  # Install GitOps in the %s namespace
  gitops install --config-repo=ssh://git@github.com/me/mygitopsrepo.git`, wego.DefaultNamespace),
	RunE:          installRunCmd,
	SilenceErrors: true,
	SilenceUsage:  true,
	PostRun: func(cmd *cobra.Command, args []string) {
		version.CheckVersion(version.CheckpointParamsWithFlags(version.CheckpointParams(), cmd))
	},
}

func init() {
	Cmd.Flags().BoolVar(&installParams.DryRun, "dry-run", false, "Outputs all the manifests that would be installed")
	Cmd.Flags().BoolVar(&installParams.AutoMerge, "auto-merge", false, "If set, 'gitops install' will automatically update the default branch for the configuration repository")
	Cmd.Flags().StringVar(&installParams.ConfigRepo, "config-repo", "", "URL of external repository that will hold automation manifests")
	cobra.CheckErr(Cmd.MarkFlagRequired("config-repo"))
}

func installRunCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	namespace, _ := cmd.Parent().Flags().GetString("namespace")

	configURL, err := gitproviders.NewRepoURL(installParams.ConfigRepo)
	if err != nil {
		return err
	}

	osysClient := osys.New()
	fluxClient := flux.New(osysClient, &runner.CLIRunner{})

	kubeClient, rawK8sClient, err := kube.NewKubeHTTPClient()
	if err != nil {
		return fmt.Errorf("error creating k8s http client: %w", err)
	}

	clusterName, err := kubeClient.GetClusterName(ctx)
	if err != nil {
		return err
	}

	log := internal.NewCLILogger(os.Stdout)

	token, err := internal.GetToken(configURL, osysClient.Stdout(), osysClient.LookupEnv, auth.NewAuthCLIHandler, log)
	if err != nil {
		return err
	}

	gitProvider, err := gitproviders.New(gitproviders.Config{
		Provider: configURL.Provider(),
		Token:    token,
		Hostname: configURL.URL().Host,
	}, configURL.Owner(), gitproviders.GetAccountType)
	if err != nil {
		return fmt.Errorf("error creating git provider client: %w", err)
	}

	if installParams.DryRun {
		manifests, err := models.BootstrapManifests(ctx, fluxClient, gitProvider, kubeClient, models.ManifestsParams{
			ClusterName:   clusterName,
			WegoNamespace: namespace,
			ConfigRepo:    configURL,
		})
		if err != nil {
			return fmt.Errorf("failed getting gitops manifests: %w", err)
		}

		for _, manifest := range manifests {
			fmt.Println(string(manifest.Content))
		}

		return nil
	}

	namespaceObj := &corev1.Namespace{}
	namespaceObj.Name = namespace

	if err := rawK8sClient.Create(ctx, namespaceObj); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed creating namespace %s: %w", namespace, err)
		}
	}

	authService, err := auth.NewAuthService(fluxClient, rawK8sClient, gitProvider, log)
	if err != nil {
		return err
	}

	deployKey, err := authService.SetupDeployKey(ctx, namespace, configURL)
	if err != nil {
		return err
	}

	gitClient := git.New(deployKey, wrapper.NewGoGit())

	repoWriter := gitopswriter.NewRepoWriter(log, gitClient, gitProvider)
	installer := install.NewInstaller(fluxClient, kubeClient, gitClient, gitProvider, log, repoWriter)

	if err = installer.Install(namespace, configURL, installParams.AutoMerge); err != nil {
		return fmt.Errorf("failed installing: %w", err)
	}

	return nil
}
