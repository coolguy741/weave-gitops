package flux

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var testFluxLogResponse = []byte(`2021-04-12T19:53:58.545Z info Alert - Starting EventSource 
2021-04-12T19:53:58.545Z info Receiver - Starting EventSource 
2021-04-12T19:53:58.545Z info Provider - Starting EventSource 
2021-04-12T19:53:58.646Z info Alert - Starting Controller 
2021-04-12T19:53:58.647Z info Alert - Starting workers 
2021-04-12T19:53:58.652Z info Provider - Starting Controller 
2021-04-12T19:53:58.652Z info Provider - Starting workers 
2021-04-12T19:54:23.373Z info GitRepository/flux-system.flux-system - Discarding event, no alerts found for the involved object 
2021-04-13T20:37:20.565Z info Kustomization/flux-system.flux-system - Discarding event, no alerts found for the involved object 
2021-04-13T20:37:21.213Z info GitRepository/podinfo.flux-system - Discarding event, no alerts found for the involved object 
2021-04-13T20:39:30.367Z info GitRepository/flux-system.flux-system - Discarding event, no alerts found for the involved object 

2021-04-12T19:54:02.383Z info  - metrics server is starting to listen 
2021-04-12T19:54:02.384Z info  - starting manager 
2021-04-12T19:54:02.385Z info  - starting metrics server 
2021-04-12T19:54:02.486Z info  - starting file server 
2021-04-12T19:54:02.486Z info HelmRepository - Starting EventSource 
2021-04-12T19:54:02.486Z info Bucket - Starting EventSource 
2021-04-12T19:54:02.486Z info HelmChart - Starting EventSource 
2021-04-12T19:54:02.486Z info HelmChart - Starting EventSource 
2021-04-12T19:54:02.487Z info HelmChart - Starting EventSource 
2021-04-12T19:54:02.587Z info GitRepository - Starting workers 
2021-04-12T19:54:02.588Z info HelmChart - Starting Controller 
2021-04-12T19:54:02.588Z info HelmRepository - Starting workers 
2021-04-12T19:54:02.588Z info HelmChart - Starting workers 
2021-04-12T19:54:02.588Z info Bucket - Starting Controller 
2021-04-12T19:54:02.589Z info Bucket - Starting workers 
2021-04-12T21:02:22.808Z info GitRepository/flux-system.flux-system - Reconciliation finished in 873.5428ms, next run in 1m0s 
2021-04-12T21:03:23.646Z info GitRepository/flux-system.flux-system - Reconciliation finished in 907.3404ms, next run in 1m0s`)

func TestGetLogs(t *testing.T) {
	result, err := getLastLogForNamespaces(testFluxLogResponse)
	require.NoError(t, err)
	require.Equal(t, 11, len(result))

	emptyResult, err := getLastLogForNamespaces(nil)
	require.NoError(t, err)
	require.Equal(t, 0, len(emptyResult))

}
