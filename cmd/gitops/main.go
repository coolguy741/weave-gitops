package main

import (
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	wego "github.com/weaveworks/weave-gitops/api/v1alpha1"
	"github.com/weaveworks/weave-gitops/cmd/gitops/add"
	"github.com/weaveworks/weave-gitops/cmd/gitops/app"
	beta "github.com/weaveworks/weave-gitops/cmd/gitops/beta/cmd"
	"github.com/weaveworks/weave-gitops/cmd/gitops/delete"
	"github.com/weaveworks/weave-gitops/cmd/gitops/docs"
	"github.com/weaveworks/weave-gitops/cmd/gitops/flux"
	"github.com/weaveworks/weave-gitops/cmd/gitops/get"
	"github.com/weaveworks/weave-gitops/cmd/gitops/install"
	"github.com/weaveworks/weave-gitops/cmd/gitops/resume"
	"github.com/weaveworks/weave-gitops/cmd/gitops/suspend"
	"github.com/weaveworks/weave-gitops/cmd/gitops/ui"
	"github.com/weaveworks/weave-gitops/cmd/gitops/uninstall"
	"github.com/weaveworks/weave-gitops/cmd/gitops/upgrade"
	"github.com/weaveworks/weave-gitops/cmd/gitops/version"
	fluxBin "github.com/weaveworks/weave-gitops/pkg/flux"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/osys"
	"github.com/weaveworks/weave-gitops/pkg/runner"
	"github.com/weaveworks/weave-gitops/pkg/utils"
	"k8s.io/client-go/rest"
)

var options struct {
	endpoint          string
	overrideInCluster bool
	verbose           bool
}

var rootCmd = &cobra.Command{
	Use:           "gitops",
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         "Weave GitOps",
	Long:          "Command line utility for managing Kubernetes applications via GitOps.",
	Example: fmt.Sprintf(`
  # Get verbose output for any gitops command
  gitops [command] -v, --verbose

  # Get gitops app help
  gitops help app

  # Add application to gitops control from a local git repository
  gitops add app . --name <myapp>
  OR
  gitops add app <myapp-directory>

  # Add application to gitops control from a github repository
  gitops add app \
    --name <myapp> \
    --url git@github.com:myorg/<myapp> \
    --branch prod-<myapp>

  # Get status of application under gitops control
  gitops app status podinfo

  # Get help for gitops add app command
  gitops add app -h
  gitops help add app

  # Show manifests that would be installed by the gitops install command
  gitops install --dry-run

  # Install gitops in the %s namespace
  gitops install

  # Get the version of gitops along with commit, branch, and flux version
  gitops version

  To learn more, you can find our documentation at https://docs.gitops.weave.works/
`, wego.DefaultNamespace),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configureLogger()

		ns, _ := cmd.Flags().GetString("namespace")

		if ns == "" {
			return
		}

		if nserr := utils.ValidateNamespace(ns); nserr != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", nserr)
			os.Exit(1)
		}
		if options.overrideInCluster {
			kube.InClusterConfig = func() (*rest.Config, error) { return nil, rest.ErrNotInCluster }
		}
	},
}

func configureLogger() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if options.verbose {
		log.SetLevel(log.DebugLevel)
	}
}

func main() {
	restyClient := resty.New()
	cliRunner := &runner.CLIRunner{}
	osysClient := osys.New()
	fluxClient := fluxBin.New(osysClient, cliRunner)
	fluxClient.SetupBin()
	rootCmd.PersistentFlags().BoolVarP(&options.verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("namespace", wego.DefaultNamespace, "Weave GitOps runtime namespace")
	rootCmd.PersistentFlags().StringVarP(&options.endpoint, "endpoint", "e", os.Getenv("WEAVE_GITOPS_ENTERPRISE_API_URL"), "The Weave GitOps Enterprise HTTP API endpoint")
	rootCmd.PersistentFlags().BoolVar(&options.overrideInCluster, "override-in-cluster", false, "override running in cluster check")
	cobra.CheckErr(rootCmd.PersistentFlags().MarkHidden("override-in-cluster"))

	rootCmd.AddCommand(install.Cmd)
	rootCmd.AddCommand(beta.Cmd)
	rootCmd.AddCommand(uninstall.Cmd)
	rootCmd.AddCommand(version.Cmd)
	rootCmd.AddCommand(flux.Cmd)
	rootCmd.AddCommand(ui.Cmd)
	rootCmd.AddCommand(app.ApplicationCmd)
	rootCmd.AddCommand(get.GetCommand(&options.endpoint, restyClient))
	rootCmd.AddCommand(add.GetCommand(&options.endpoint, restyClient))
	rootCmd.AddCommand(delete.DeleteCommand(&options.endpoint, restyClient))
	rootCmd.AddCommand(resume.GetCommand())
	rootCmd.AddCommand(suspend.GetCommand())
	rootCmd.AddCommand(upgrade.Cmd)
	rootCmd.AddCommand(docs.Cmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
