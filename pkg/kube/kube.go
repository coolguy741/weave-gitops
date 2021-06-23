package kube

import (
	"bytes"
	"fmt"
	"strings"

	"encoding/json"

	"github.com/pkg/errors"
	wego "github.com/weaveworks/weave-gitops/api/v1alpha"
	"github.com/weaveworks/weave-gitops/pkg/runner"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

const kubectlPath = "kubectl"

type ClusterStatus int

const (
	Unknown ClusterStatus = iota
	Unmodified
	FluxInstalled
	WeGOInstalled
)

// Function to translate ClusterStatus to a string
func (cs ClusterStatus) String() string {
	return toStatusString[cs]
}

var toStatusString = map[ClusterStatus]string{
	Unknown:       "Unknown",
	Unmodified:    "Unmodified",
	FluxInstalled: "FluxInstalled",
	WeGOInstalled: "WeGOInstalled",
}

//counterfeiter:generate . Kube
type Kube interface {
	Apply(manifests []byte, namespace string) ([]byte, error)
	GetClusterName() (string, error)
	GetClusterStatus() ClusterStatus
	GetApplication(name string) (*wego.Application, error)
}

type KubeClient struct {
	runner runner.Runner
}

func New(cliRunner runner.Runner) *KubeClient {
	return &KubeClient{
		runner: cliRunner,
	}
}

var _ Kube = &KubeClient{}

func (k *KubeClient) Apply(manifests []byte, namespace string) ([]byte, error) {
	args := []string{
		"apply",
		"--namespace", namespace,
		"-f", "-",
	}

	out, err := k.runKubectlCmdWithInput(args, manifests)
	if err != nil {
		return out, err
	}

	return out, nil
}

func (k *KubeClient) GetClusterName() (string, error) {
	args := []string{
		"config", "current-context",
	}

	out, err := k.runKubectlCmd(args)
	if err != nil {
		return "", errors.Wrap(err, "failed to get kubectl current-context")
	}

	return string(bytes.TrimSuffix(out, []byte("\n"))), nil
}

func (k *KubeClient) GetClusterStatus() ClusterStatus {
	// Checking wego presence
	if k.resourceLookup("get crd apps.wego.weave.works") == nil {
		return WeGOInstalled
	}

	// Checking flux presence
	if k.resourceLookup("get namespace flux-system") == nil {
		return FluxInstalled
	}

	if k.resourceLookup("deployment coredns -n kube-system") == nil {
		return Unmodified
	}

	return Unknown
}

func (k *KubeClient) GetApplication(name string) (*wego.Application, error) {
	cmd := []string{"get", "app", name, "-o", "json"}
	o, err := k.runKubectlCmd(cmd)

	if err != nil {
		return nil, fmt.Errorf("could not run kubectl command: %s", err)
	}

	a := wego.Application{}

	if err := json.Unmarshal(o, &a); err != nil {
		return nil, fmt.Errorf("could not unmarshal json: %s", err)
	}

	return &a, nil
}

func (k *KubeClient) resourceLookup(args string) error {
	_, err := k.runKubectlCmd(strings.Split(args, " "))
	if err != nil {
		return err
	}

	return nil
}

func (k *KubeClient) runKubectlCmd(args []string) ([]byte, error) {
	out, err := k.runner.Run(kubectlPath, args...)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to run kubectl with output: %s", string(out))
	}

	return out, nil
}

func (k *KubeClient) runKubectlCmdWithInput(args []string, input []byte) ([]byte, error) {
	out, err := k.runner.RunWithStdin(kubectlPath, args, input)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to run kubectl with output: %s", string(out))
	}

	return out, nil
}
