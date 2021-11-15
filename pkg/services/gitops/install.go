package gitops

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"github.com/weaveworks/weave-gitops/cmd/gitops/version"
	"github.com/weaveworks/weave-gitops/manifests"
	"github.com/weaveworks/weave-gitops/pkg/git"
	"github.com/weaveworks/weave-gitops/pkg/gitproviders"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/services/automation"
	"github.com/weaveworks/weave-gitops/pkg/services/gitrepo"
)

type InstallParams struct {
	Namespace    string
	DryRun       bool
	AppConfigURL string
}

func (g *Gitops) Install(params InstallParams) (map[string][]byte, error) {
	ctx := context.Background()
	status := g.kube.GetClusterStatus(ctx)

	switch status {
	case kube.FluxInstalled:
		return nil, errors.New("Weave GitOps does not yet support installation onto a cluster that is using Flux.\nPlease uninstall flux before proceeding:\n  $ flux uninstall")
	case kube.Unknown:
		return nil, errors.New("Weave GitOps cannot talk to the cluster")
	}

	// TODO apply these manifests instead of generating them again
	var fluxManifests []byte

	var err error

	if params.AppConfigURL != "" || params.DryRun {
		// We need to get the manifests to persist in the repo and
		// non-dry run install doesn't return them
		fluxManifests, err = g.flux.Install(params.Namespace, true)
		if err != nil {
			return nil, fmt.Errorf("error on flux install %w", err)
		}
	}

	if !params.DryRun {
		_, err = g.flux.Install(params.Namespace, false)
		if err != nil {
			return nil, fmt.Errorf("error on flux install %w", err)
		}
	}

	systemManifests := make(map[string][]byte)
	systemManifests["gitops-runtime.yaml"] = fluxManifests
	systemManifests["wego-system.yaml"] = manifests.AppCRD

	if params.DryRun {
		return systemManifests, nil
	} else {
		if err := g.kube.Apply(ctx, manifests.AppCRD, params.Namespace); err != nil {
			return nil, fmt.Errorf("could not apply App CRD: %w", err)
		}

		version := version.Version
		if os.Getenv("IS_TEST_ENV") != "" {
			version = "latest"
		}

		wegoAppManifests, err := manifests.GenerateWegoAppManifests(manifests.WegoAppParams{Version: version, Namespace: params.Namespace})
		if err != nil {
			return nil, fmt.Errorf("error generating wego-app manifests, %w", err)
		}
		for _, m := range wegoAppManifests {
			if err := g.kube.Apply(ctx, m, params.Namespace); err != nil {
				return nil, fmt.Errorf("error applying wego-app manifest %s: %w", m, err)
			}
		}

		systemManifests["wego-app.yaml"] = bytes.Join(wegoAppManifests, []byte("---\n"))
	}

	return systemManifests, nil
}

func (g *Gitops) StoreManifests(gitClient git.Git, gitProvider gitproviders.GitProvider, params InstallParams, systemManifests map[string][]byte) (map[string][]byte, error) {
	ctx := context.Background()

	if !params.DryRun && params.AppConfigURL != "" {
		cname, err := g.kube.GetClusterName(ctx)
		if err != nil {
			g.logger.Warningf("Cluster name not found, using default : %v", err)

			cname = "default"
		}

		goatManifests, err := g.storeManifests(gitClient, gitProvider, params, systemManifests, cname)
		if err != nil {
			return nil, fmt.Errorf("failed to store cluster manifests: %w", err)
		}

		g.logger.Actionf("Applying manifests to the cluster")
		// only apply the system manifests as the others will get picked up once flux is running
		if err := g.applyManifestsToK8s(ctx, params.Namespace, goatManifests); err != nil {
			return nil, fmt.Errorf("failed applying system manifests to cluster %s :%w", cname, err)
		}
	}

	return systemManifests, nil
}

func (g *Gitops) storeManifests(gitClient git.Git, gitProvider gitproviders.GitProvider, params InstallParams, systemManifests map[string][]byte, cname string) (map[string][]byte, error) {
	ctx := context.Background()

	normalizedURL, err := gitproviders.NewRepoURL(params.AppConfigURL)
	if err != nil {
		return nil, fmt.Errorf("failed to convert app config repo %q : %w", params.AppConfigURL, err)
	}

	configBranch, err := gitProvider.GetDefaultBranch(ctx, normalizedURL)
	if err != nil {
		return nil, fmt.Errorf("could not determine default branch for config repository: %q %w", params.AppConfigURL, err)
	}

	remover, _, err := gitrepo.CloneRepo(ctx, gitClient, normalizedURL, configBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to clone configuration repo: %w", err)
	}

	defer remover()

	manifests := make(map[string][]byte, 3)
	clusterPath := filepath.Join(git.WegoRoot, git.WegoClusterDir, cname)

	gitsource, sourceName, err := g.genSource(configBranch, params.Namespace, normalizedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create source manifest: %w", err)
	}
	// filepath.Join doesn't support a starting "."
	prefixForFlux := func(s string) string {
		return "." + s
	}
	manifests["flux-source-resource.yaml"] = gitsource

	system, err := g.genKustomize(automation.ConstrainResourceName(fmt.Sprintf("%s-system", cname)), sourceName,
		prefixForFlux(filepath.Join(".", clusterPath, git.WegoClusterOSWorkloadDir)), params)
	if err != nil {
		return nil, fmt.Errorf("failed to create system kustomization manifest: %w", err)
	}

	manifests["flux-system-kustomization-resource.yaml"] = system

	user, err := g.genKustomize(automation.ConstrainResourceName(fmt.Sprintf("%s-user", cname)), sourceName,
		prefixForFlux(filepath.Join(".", clusterPath, git.WegoClusterUserWorloadDir)), params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user kustomization manifest: %w", err)
	}

	manifests["flux-user-kustomization-resource.yaml"] = user

	//TODO add handling for PRs
	// if !params.AutoMerge {
	//  if err := a.createPullRequestToRepo(info, info.Spec.ConfigURL, appHash, appSpec, appGoat, appSource); err != nil {
	//      return err
	//  }
	// } else {
	g.logger.Actionf("Writing manifests to disk")

	if err := g.writeManifestsToGit(gitClient, filepath.Join(clusterPath, "system"), manifests); err != nil {
		return nil, fmt.Errorf("failed to write manifests: %w", err)
	}

	if err := g.writeManifestsToGit(gitClient, filepath.Join(clusterPath, "system"), systemManifests); err != nil {
		return nil, fmt.Errorf("failed to write system manifests: %w", err)
	}
	// store a .keep file in the user dir
	userKeep := map[string][]byte{
		".keep": strconv.AppendQuote(nil, "# keep"),
	}
	if err := g.writeManifestsToGit(gitClient, filepath.Join(clusterPath, "user"), userKeep); err != nil {
		return nil, fmt.Errorf("failed to write user manifests: %w", err)
	}

	return manifests, gitrepo.CommitAndPush(ctx, gitClient, "Add GitOps runtime manifests", g.logger)
}

func (g *Gitops) genSource(branch string, namespace string, normalizedUrl gitproviders.RepoURL) ([]byte, string, error) {
	secretRef := automation.CreateRepoSecretName(normalizedUrl).String()

	sourceManifest, err := g.flux.CreateSourceGit(secretRef, normalizedUrl, branch, secretRef, namespace)
	if err != nil {
		return nil, secretRef, fmt.Errorf("could not create git source for repo %s : %w", normalizedUrl.String(), err)
	}

	return sourceManifest, secretRef, nil
}

func (g *Gitops) genKustomize(name, cname, path string, params InstallParams) ([]byte, error) {
	sourceManifest, err := g.flux.CreateKustomization(name, cname, path, params.Namespace)
	if err != nil {
		return nil, fmt.Errorf("could not create flux kustomization for path %q : %w", path, err)
	}

	return sourceManifest, nil
}

func (g *Gitops) writeManifestsToGit(gitClient git.Git, path string, manifests map[string][]byte) error {
	for k, m := range manifests {
		if err := gitClient.Write(filepath.Join(path, k), m); err != nil {
			g.logger.Warningf("failed to write manifest %s : %v", k, err)
			return err
		}
	}

	return nil
}

func (g *Gitops) applyManifestsToK8s(ctx context.Context, namespace string, manifests map[string][]byte) error {
	for k, manifest := range manifests {
		if err := g.kube.Apply(ctx, manifest, namespace); err != nil {
			return fmt.Errorf("could not apply manifest %q : %w", k, err)
		}
	}

	return nil
}
