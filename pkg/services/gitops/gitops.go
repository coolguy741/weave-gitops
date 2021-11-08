package gitops

import (
	"github.com/weaveworks/weave-gitops/pkg/flux"
	"github.com/weaveworks/weave-gitops/pkg/git"
	"github.com/weaveworks/weave-gitops/pkg/gitproviders"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/logger"
)

type GitopsService interface {
	Install(params InstallParams) (map[string][]byte, error)
	StoreManifests(gitClient git.Git, gitProvider gitproviders.GitProvider, params InstallParams, systemManifests map[string][]byte) (map[string][]byte, error)
	Uninstall(params UninstallParams) error
}

type Gitops struct {
	flux   flux.Flux
	kube   kube.Kube
	logger logger.Logger
}

func New(logger logger.Logger, flux flux.Flux, kube kube.Kube) GitopsService {
	return &Gitops{
		flux:   flux,
		kube:   kube,
		logger: logger,
	}
}
