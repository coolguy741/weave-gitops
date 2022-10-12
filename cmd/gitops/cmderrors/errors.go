package cmderrors

import "errors"

var (
	ErrNoWGEEndpoint          = errors.New("the Weave GitOps Enterprise HTTP API endpoint flag (--endpoint) has not been set")
	ErrNoURL                  = errors.New("the URL flag (--url) has not been set")
	ErrNoTLSCertOrKey         = errors.New("flags --tls-cert-file and --tls-private-key-file cannot be empty")
	ErrNoFilePath             = errors.New("the filepath has not been set")
	ErrMultipleFilePaths      = errors.New("only one filepath is allowed")
	ErrInvalidArgs            = errors.New("invalid positional arguments")
	ErrNoContextForKubeConfig = errors.New("no context provided for the kubeconfig")
	ErrNoCluster              = errors.New("no cluster in the kube config")
	ErrGetKubeClient          = errors.New("error getting Kube HTTP client")
	ErrNoName                 = errors.New("name is required")
	ErrMultipleNames          = errors.New("only one name is allowed")
)
