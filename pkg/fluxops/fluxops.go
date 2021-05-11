package fluxops

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/weaveworks/weave-gitops/pkg/shims"
	"github.com/weaveworks/weave-gitops/pkg/status"
	"github.com/weaveworks/weave-gitops/pkg/utils"
	"github.com/weaveworks/weave-gitops/pkg/version"
	"sigs.k8s.io/yaml"
)

const fluxSystemNamespace = `apiVersion: v1
kind: Namespace
metadata:
  name: flux-system
---
`

var (
	fluxHandler FluxHandler = defaultFluxHandler{}
	fluxBinary  string
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . FluxHandler
type FluxHandler interface {
	Handle(args string) ([]byte, error)
}

type defaultFluxHandler struct{}

func (h defaultFluxHandler) Handle(arglist string) ([]byte, error) {
	initFluxBinary()
	return utils.CallCommand(fmt.Sprintf("%s %s", fluxBinary, arglist))
}

type quietFluxHandler struct{}

func (q quietFluxHandler) Handle(arglist string) ([]byte, error) {
	initFluxBinary()
	return utils.CallCommandSilently(fmt.Sprintf("%s %s", fluxBinary, arglist))
}

// WithFluxHandler allows running a function with a different flux handler in force
func WithFluxHandler(handler FluxHandler, f func() ([]byte, error)) ([]byte, error) {
	switch fluxHandler.(type) {
	case defaultFluxHandler:
		existingHandler := fluxHandler
		fluxHandler = handler
		defer func() {
			fluxHandler = existingHandler
		}()
		return f()
	default:
		return f()
	}
}

func FluxPath() (string, error) {
	homeDir, err := shims.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := fmt.Sprintf("%v/.wego/bin", homeDir)
	return fmt.Sprintf("%v/flux-%v", path, version.FluxVersion), nil
}

func SetFluxHandler(h FluxHandler) {
	fluxHandler = h
}

func CallFlux(arglist ...string) ([]byte, error) {
	return fluxHandler.Handle(strings.Join(arglist, " "))
}

func Install(namespace string) ([]byte, error) {
	return installFlux(namespace, true)
}

func QuietInstall(namespace string) ([]byte, error) {
	return installFlux(namespace, false)
}

func installFlux(namespace string, verbose bool) ([]byte, error) {
	var extraManifest []byte
	if namespace != "flux-system" { // we need to have this namespace created
		extraManifest = []byte(fluxSystemNamespace)
	}

	args := []string{
		"install",
		fmt.Sprintf("--namespace=%s", namespace),
		"--export",
	}

	if verbose {
		manifests, err := CallFlux(args...)
		if err != nil {
			return nil, err
		}
		return append(extraManifest, manifests...), nil
	}

	return WithFluxHandler(quietFluxHandler{}, func() ([]byte, error) {
		manifests, err := CallFlux(args...)
		if err != nil {
			return nil, err
		}
		return append(extraManifest, manifests...), nil
	})
}

// GetOwnerFromEnv determines the owner of a new repository based on the GITHUB_ORG
func GetOwnerFromEnv() (string, error) {
	// check for github username
	user, okUser := os.LookupEnv("GITHUB_ORG")
	if okUser {
		return user, nil
	}

	return GetUserFromHubCredentials()
}

// GetRepoName returns the name of the wego repo for the cluster (the repo holding controller defs)
func GetRepoName() (string, error) {
	clusterName, err := status.GetClusterName()
	if err != nil {
		return "", err
	}
	return clusterName + "-wego", nil
}

func GetUserFromHubCredentials() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// check for existing ~/.config/hub
	config, err := ioutil.ReadFile(filepath.Join(homeDir, ".config", "hub"))
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{}
	err = yaml.Unmarshal(config, &data)
	if err != nil {
		return "", err
	}

	return data["github.com"].([]interface{})[0].(map[string]interface{})["user"].(string), nil
}

func initFluxBinary() {
	if fluxBinary == "" {
		fluxPath, err := FluxPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to retrieve wego executable path: %v", err)
			shims.Exit(1)
		}
		fluxBinary = fluxPath
	}
}
