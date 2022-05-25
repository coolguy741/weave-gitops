package terraform

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/weaveworks/weave-gitops/cmd/gitops/cmderrors"
	"github.com/weaveworks/weave-gitops/cmd/internal"
	"github.com/weaveworks/weave-gitops/pkg/adapters"
	"github.com/weaveworks/weave-gitops/pkg/gitproviders"
	"github.com/weaveworks/weave-gitops/pkg/templates"
)

type terraformCommandFlags struct {
	Template              string
	ParameterValues       []string
	RepositoryURL         string
	BaseBranch            string
	HeadBranch            string
	Title                 string
	Description           string
	CommitMessage         string
	InsecureSkipTlsVerify bool
}

var flags terraformCommandFlags

func AddCommand(endpoint, username, password *string, client *resty.Client) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terraform",
		Short: "Add a new Terraform resource using a TF template",
		Example: `
# Add a new Terraform resource using a TF template
gitops add terraform --from-template <template-name> --set key=val
		`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE:       addTerraformCmdPreRunE(endpoint, client),
		RunE:          addTerraformCmdRunE(endpoint, username, password, client),
	}

	cmd.Flags().StringVar(&flags.RepositoryURL, "url", "", "URL of remote repository to create the pull request")
	internal.AddTemplateFlags(cmd, &flags.Template, &flags.ParameterValues)
	internal.AddPRFlags(cmd, &flags.HeadBranch, &flags.BaseBranch, &flags.Description, &flags.CommitMessage, &flags.Title)

	return cmd
}

func addTerraformCmdPreRunE(endpoint *string, client *resty.Client) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, s []string) error {
		if *endpoint == "" {
			return cmderrors.ErrNoWGEEndpoint
		}

		return nil
	}
}

func addTerraformCmdRunE(endpoint, username, password *string, client *resty.Client) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		r, err := adapters.NewHttpClient(*endpoint, *username, *password, client, os.Stdout)
		if err != nil {
			return err
		}

		vals := make(map[string]string)

		for _, v := range flags.ParameterValues {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) == 2 {
				vals[kv[0]] = kv[1]
			}
		}

		if flags.RepositoryURL == "" {
			return cmderrors.ErrNoURL
		}

		url, err := gitproviders.NewRepoURL(flags.RepositoryURL)
		if err != nil {
			return fmt.Errorf("cannot parse url: %w", err)
		}

		token, err := internal.GetToken(url, os.LookupEnv)
		if err != nil {
			return err
		}

		params := templates.CreatePullRequestFromTemplateParams{
			GitProviderToken: token,
			TemplateName:     flags.Template,
			TemplateKind:     templates.GitopsTemplateKind,
			ParameterValues:  vals,
			RepositoryURL:    flags.RepositoryURL,
			HeadBranch:       flags.HeadBranch,
			BaseBranch:       flags.BaseBranch,
			Title:            flags.Title,
			Description:      flags.Description,
			CommitMessage:    flags.CommitMessage,
		}

		return templates.CreatePullRequestFromTemplate(params, r, os.Stdout)
	}
}
