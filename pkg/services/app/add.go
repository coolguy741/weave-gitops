package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/weaveworks/weave-gitops/pkg/git"
	"github.com/weaveworks/weave-gitops/pkg/gitproviders"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/models"
	"github.com/weaveworks/weave-gitops/pkg/services/automation"
	"github.com/weaveworks/weave-gitops/pkg/services/gitopswriter"
	"github.com/weaveworks/weave-gitops/pkg/services/gitrepo"
	"github.com/weaveworks/weave-gitops/pkg/utils"

	wego "github.com/weaveworks/weave-gitops/api/v1alpha1"
)

const (
	WeGOAppIdentifierLabelKey = "wego.weave.works/app-identifier"
)

type AddParams struct {
	Dir                        string
	Name                       string
	Url                        string
	Path                       string
	Branch                     string
	DeploymentType             string
	Chart                      string
	SourceType                 wego.SourceType
	AppConfigUrl               string
	Namespace                  string
	DryRun                     bool
	AutoMerge                  bool
	GitProviderToken           string
	HelmReleaseTargetNamespace string
	MigrateToNewDirStructure   func(string) string
}

const (
	DefaultPath           = "./"
	DefaultBranch         = "main"
	DefaultDeploymentType = "kustomize"
)

func (a AddParams) IsHelmRepository() bool {
	return a.Chart != ""
}

func (a *AppSvc) Add(configGit git.Git, gitProvider gitproviders.GitProvider, params AddParams) error {
	ctx := context.Background()

	params, err := a.updateParametersIfNecessary(ctx, gitProvider, params)
	if err != nil {
		return fmt.Errorf("could not update parameters: %w", err)
	}

	a.printAddSummary(params)

	if err := kube.IsClusterReady(a.Logger, a.Kube); err != nil {
		return err
	}

	clusterName, err := a.Kube.GetClusterName(ctx)
	if err != nil {
		return err
	}

	app, err := makeApplication(params)
	if err != nil {
		return err
	}

	appHash := automation.GetAppHash(app)

	wegoapps, err := a.Kube.GetApplications(ctx, params.Namespace)
	if err != nil {
		return err
	}

	for _, wegoapp := range wegoapps {
		clusterApp, err := automation.WegoAppToApp(wegoapp)
		if err != nil {
			return err
		}

		if appHash == automation.GetAppHash(clusterApp) {
			return fmt.Errorf("unable to create resource, resource already exists in cluster")
		}
	}

	if params.DryRun {
		return nil
	}

	return a.addApp(ctx, configGit, gitProvider, app, clusterName, params.AutoMerge)
}

func (a *AppSvc) printAddSummary(params AddParams) {
	a.Logger.Println("Adding application:\n")
	a.Logger.Println("Name: %s", params.Name)
	a.Logger.Println("URL: %s", params.Url)
	a.Logger.Println("Path: %s", params.Path)
	a.Logger.Println("Branch: %s", params.Branch)
	a.Logger.Println("Type: %s", params.DeploymentType)

	if params.Chart != "" {
		a.Logger.Println("Chart: %s", params.Chart)
	}

	a.Logger.Println("")
}

func (a *AppSvc) updateParametersIfNecessary(ctx context.Context, gitProvider gitproviders.GitProvider, params AddParams) (AddParams, error) {
	params.SourceType = wego.SourceTypeGit

	var appRepoUrl gitproviders.RepoURL

	switch {
	case params.Chart != "":
		params.SourceType = wego.SourceTypeHelm
		params.DeploymentType = string(wego.DeploymentTypeHelm)
		params.Path = params.Chart

		if params.Name == "" {
			if automation.ApplicationNameTooLong(params.Chart) {
				return params, fmt.Errorf("chart name %q is too long to use as application name (must be <= %d characters); please specify name with '--name'",
					params.Chart, automation.MaxKubernetesResourceNameLength)
			}

			params.Name = params.Chart
		}

		if params.Url == "" {
			return params, fmt.Errorf("--url must be specified for helm repositories")
		}

		if params.AppConfigUrl == "" {
			return params, errors.New("--app-config-url should be provided")
		}
	default:
		var err error

		appRepoUrl, err = gitproviders.NewRepoURL(params.Url)
		if err != nil {
			return params, fmt.Errorf("error normalizing url: %w", err)
		}

		params.Url = appRepoUrl.String()

		// resetting Dir param since Url has priority over it
		params.Dir = ""
	}

	// making sure the config url is in good format
	if models.IsExternalConfigUrl(params.AppConfigUrl) {
		configRepoUrl, err := gitproviders.NewRepoURL(params.AppConfigUrl)
		if err != nil {
			return params, fmt.Errorf("error normalizing url: %w", err)
		}

		params.AppConfigUrl = configRepoUrl.String()
	}

	if params.Name == "" {
		repoName := utils.UrlToRepoName(params.Url)
		if automation.ApplicationNameTooLong(repoName) {
			return params, fmt.Errorf("url base name %q is too long to use as application name (must be <= %d characters); please specify name with '--name'",
				repoName, automation.MaxKubernetesResourceNameLength)
		}

		params.Name = automation.GenerateResourceName(appRepoUrl)
	}

	if params.Path == "" {
		params.Path = DefaultPath
	}

	if params.DeploymentType == "" {
		params.DeploymentType = DefaultDeploymentType
	}

	if params.Branch == "" {
		params.Branch = DefaultBranch

		if params.SourceType == wego.SourceTypeGit {
			branch, err := gitProvider.GetDefaultBranch(ctx, appRepoUrl)
			if err != nil {
				return params, err
			} else {
				params.Branch = branch
			}
		}
	}

	if automation.ApplicationNameTooLong(params.Name) {
		return params, fmt.Errorf("application name too long: %s; must be <= %d characters", params.Name, automation.MaxKubernetesResourceNameLength)
	}

	// Validate namespace argument for helm
	if params.HelmReleaseTargetNamespace != "" {
		if ok, _ := a.Kube.NamespacePresent(context.Background(), params.HelmReleaseTargetNamespace); !ok {
			return params, fmt.Errorf("Helm Release Target Namespace %s does not exist", params.HelmReleaseTargetNamespace)
		}

		if nserr := utils.ValidateNamespace(params.HelmReleaseTargetNamespace); nserr != nil {
			return params, nserr
		}
	}

	return params, nil
}

func (a *AppSvc) addApp(ctx context.Context, configGit git.Git, gitProvider gitproviders.GitProvider, app models.Application, clusterName string, autoMerge bool) error {
	repoWriter := gitrepo.NewRepoWriter(app.ConfigURL, gitProvider, configGit, a.Logger)
	automationGen := automation.NewAutomationGenerator(gitProvider, a.Flux, a.Logger)
	gitOpsDirWriter := gitopswriter.NewGitOpsDirectoryWriter(automationGen, repoWriter, a.Osys, a.Logger)

	return gitOpsDirWriter.AddApplication(ctx, app, clusterName, autoMerge)
}

func makeApplication(params AddParams) (models.Application, error) {
	var (
		gitSourceURL  gitproviders.RepoURL
		helmSourceURL string
		err           error
	)

	if models.SourceType(params.SourceType) == models.SourceTypeHelm {
		helmSourceURL = params.Url
	} else {
		gitSourceURL, err = gitproviders.NewRepoURL(params.Url)
		if err != nil {
			return models.Application{}, err
		}
	}

	configURL := gitSourceURL

	if params.AppConfigUrl != "" {
		curl, err := gitproviders.NewRepoURL(params.AppConfigUrl)
		if err != nil {
			return models.Application{}, err
		}

		configURL = curl
	}

	app := models.Application{
		Name:                params.Name,
		Namespace:           params.Namespace,
		GitSourceURL:        gitSourceURL,
		HelmSourceURL:       helmSourceURL,
		ConfigURL:           configURL,
		Branch:              params.Branch,
		Path:                params.Path,
		SourceType:          models.SourceType(params.SourceType),
		AutomationType:      models.AutomationType(params.DeploymentType),
		HelmTargetNamespace: params.HelmReleaseTargetNamespace,
	}

	return app, nil
}
