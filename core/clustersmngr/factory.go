package clustersmngr

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/cheshir/ttlcache"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/weaveworks/weave-gitops/core/nsaccess"
	"github.com/weaveworks/weave-gitops/pkg/server/auth"
	v1 "k8s.io/api/core/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/cli-utils/pkg/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

const (
	userNamespaceTTL = 30 * time.Second
	// How often we need to stop the world and remove outdated records.
	userNamespaceResolution = 30 * time.Second
	watchClustersFrequency  = 30 * time.Second
	watchNamespaceFrequency = 30 * time.Second
	kubeClientTimeout       = 8 * time.Second
	kubeClientDialTimeout   = 5 * time.Second
	kubeClientDialKeepAlive = 30 * time.Second
)

// ClientError is an error returned by the GetImpersonatedClient function which contains
// the details of the cluster that caused the error.
type ClientError struct {
	ClusterName string
	Err         error
}

// Error() returns the error message of the underlying error.
func (ce *ClientError) Error() string {
	return ce.Err.Error()
}

// ClientsFactory is a factory for creating clients for clusters
//
//counterfeiter:generate . ClientsFactory
type ClientsFactory interface {
	// GetImpersonatedClient returns the clusters client for the given user
	GetImpersonatedClient(ctx context.Context, user *auth.UserPrincipal) (Client, error)
	// GetImpersonatedClientForCluster returns the client for the given user and cluster
	GetImpersonatedClientForCluster(ctx context.Context, user *auth.UserPrincipal, clusterName string) (Client, error)
	// GetImpersonatedDiscoveryClient returns the discovery for the given user and for the given cluster
	GetImpersonatedDiscoveryClient(ctx context.Context, user *auth.UserPrincipal, clusterName string) (*discovery.DiscoveryClient, error)
	// UpdateClusters updates the clusters list
	UpdateClusters(ctx context.Context) error
	// UpdateNamespaces updates the namespaces all namespaces for all clusters
	UpdateNamespaces(ctx context.Context) error
	// UpdateUserNamespaces updates the cache of accessible namespaces for the user
	UpdateUserNamespaces(ctx context.Context, user *auth.UserPrincipal)
	// GetServerClient returns the cluster client with gitops server permissions
	GetServerClient(ctx context.Context) (Client, error)
	// GetClustersNamespaces returns the namespaces for all clusters
	GetClustersNamespaces() map[string][]v1.Namespace
	// GetUserNamespaces returns the accessible namespaces for the user
	GetUserNamespaces(user *auth.UserPrincipal) map[string][]v1.Namespace
	// Start starts go routines to keep clusters and namespaces lists up to date
	Start(ctx context.Context)
}

var DefaultKubeConfigOptions = []KubeConfigOption{WithFlowControl}

type ClusterPoolFactoryFn func(*apiruntime.Scheme) ClientsPool
type KubeConfigOption func(*rest.Config) (*rest.Config, error)

type clientsFactory struct {
	clustersFetcher ClusterFetcher
	nsChecker       nsaccess.Checker
	log             logr.Logger

	// list of clusters returned by the clusters fetcher
	clusters *Clusters
	// string containing ordered list of cluster names, used to refresh dependent caches
	clustersHash string
	// the lists of all namespaces of each cluster
	clustersNamespaces *ClustersNamespaces
	// lists of namespaces accessible by the user on every cluster
	usersNamespaces *UsersNamespaces

	initialClustersLoad chan bool
	scheme              *apiruntime.Scheme
	newClustersPool     ClusterPoolFactoryFn
	kubeConfigOptions   []KubeConfigOption
}

func NewClientFactory(fetcher ClusterFetcher, nsChecker nsaccess.Checker, logger logr.Logger, scheme *apiruntime.Scheme, clusterPoolFactory ClusterPoolFactoryFn, kubeConfigOptions []KubeConfigOption) ClientsFactory {
	return &clientsFactory{
		clustersFetcher:     fetcher,
		nsChecker:           nsChecker,
		clusters:            &Clusters{},
		clustersNamespaces:  &ClustersNamespaces{},
		usersNamespaces:     &UsersNamespaces{Cache: ttlcache.New(userNamespaceResolution)},
		log:                 logger,
		initialClustersLoad: make(chan bool),
		scheme:              scheme,
		newClustersPool:     clusterPoolFactory,
		kubeConfigOptions:   []KubeConfigOption{},
	}
}

func (cf *clientsFactory) Start(ctx context.Context) {
	go cf.watchClusters(ctx)
	go cf.watchNamespaces(ctx)
}

func (cf *clientsFactory) watchClusters(ctx context.Context) {
	if err := cf.UpdateClusters(ctx); err != nil {
		cf.log.Error(err, "failed updating clusters")
	}

	cf.initialClustersLoad <- true

	if err := wait.PollImmediateInfinite(watchClustersFrequency, func() (bool, error) {
		if err := cf.UpdateClusters(ctx); err != nil {
			cf.log.Error(err, "Failed to update clusters")
		}

		return false, nil
	}); err != nil {
		cf.log.Error(err, "failed polling clusters")
	}
}

func (cf *clientsFactory) UpdateClusters(ctx context.Context) error {
	clusters, err := cf.clustersFetcher.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch clusters: %w", err)
	}

	cf.clusters.Set(clusters)

	return nil
}

func (cf *clientsFactory) watchNamespaces(ctx context.Context) {
	// waits the first load of cluster to start watching namespaces
	<-cf.initialClustersLoad

	if err := wait.PollImmediateInfinite(watchNamespaceFrequency, func() (bool, error) {
		if err := cf.UpdateNamespaces(ctx); err != nil {
			if merr, ok := err.(*multierror.Error); ok {
				for _, cerr := range merr.Errors {
					cf.log.Error(cerr, "failed to update namespaces")
				}
			}
		}

		return false, nil
	}); err != nil {
		cf.log.Error(err, "failed polling namespaces")
	}
}

func (cf *clientsFactory) UpdateNamespaces(ctx context.Context) error {
	var result *multierror.Error

	serverClient, err := cf.GetServerClient(ctx)
	if err != nil {
		if merr, ok := err.(*multierror.Error); ok {
			for _, err := range merr.Errors {
				if cerr, ok := err.(*ClientError); ok {
					result = multierror.Append(result, fmt.Errorf("%w, cluster: %v", cerr, cerr.ClusterName))
				}
			}
		}
	}

	cf.syncCaches()

	nsList := NewClusteredList(func() client.ObjectList {
		return &v1.NamespaceList{}
	})

	if err := serverClient.ClusteredList(ctx, nsList, false); err != nil {
		result = multierror.Append(result, err)
	}

	for clusterName, lists := range nsList.Lists() {
		// This is the "namespace loop", but namespaces aren't
		// namespaced so only 1 item
		for _, l := range lists {
			list, ok := l.(*v1.NamespaceList)
			if !ok {
				continue
			}

			cf.clustersNamespaces.Set(clusterName, list.Items)
		}
	}

	return result.ErrorOrNil()
}

func (cf *clientsFactory) GetClustersNamespaces() map[string][]v1.Namespace {
	return cf.clustersNamespaces.namespaces
}

func (cf *clientsFactory) syncCaches() {
	newHash := cf.clusters.Hash()

	if newHash != cf.clustersHash {
		cf.log.Info("Clearing namespace caches")
		cf.clustersNamespaces.Clear()
		cf.usersNamespaces.Clear()
		cf.clustersHash = newHash
	}
}

func (cf *clientsFactory) GetImpersonatedClient(ctx context.Context, user *auth.UserPrincipal) (Client, error) {
	if user == nil {
		return nil, errors.New("no user supplied")
	}

	pool := cf.newClustersPool(cf.scheme)
	errChan := make(chan error, len(cf.clusters.Get()))

	var wg sync.WaitGroup

	for _, cluster := range cf.clusters.Get() {
		wg.Add(1)

		go func(cluster Cluster, pool ClientsPool, errChan chan error) {
			defer wg.Done()

			if err := pool.Add(ClientConfigWithUser(user, cf.kubeConfigOptions...), cluster); err != nil {
				errChan <- &ClientError{ClusterName: cluster.Name, Err: fmt.Errorf("failed adding cluster client to pool: %w", err)}
			}
		}(cluster, pool, errChan)
	}

	wg.Wait()
	close(errChan)

	var result *multierror.Error

	for err := range errChan {
		result = multierror.Append(result, err)
	}

	return NewClient(pool, cf.userNsList(ctx, user)), result.ErrorOrNil()
}

func (cf *clientsFactory) GetImpersonatedClientForCluster(ctx context.Context, user *auth.UserPrincipal, clusterName string) (Client, error) {
	if user == nil {
		return nil, errors.New("no user supplied")
	}

	pool := cf.newClustersPool(cf.scheme)
	clusters := cf.clusters.Get()

	var cl Cluster

	for _, c := range clusters {
		if c.Name == clusterName {
			cl = c
			break
		}
	}

	if cl.Name == "" {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	if err := pool.Add(ClientConfigWithUser(user, cf.kubeConfigOptions...), cl); err != nil {
		return nil, fmt.Errorf("failed adding cluster client to pool: %w", err)
	}

	return NewClient(pool, cf.userNsList(ctx, user)), nil
}

func (cf *clientsFactory) GetImpersonatedDiscoveryClient(ctx context.Context, user *auth.UserPrincipal, clusterName string) (*discovery.DiscoveryClient, error) {
	if user == nil {
		return nil, errors.New("no user supplied")
	}

	var config *rest.Config

	for _, cluster := range cf.clusters.Get() {
		if cluster.Name == clusterName {
			var err error

			config, err = ClientConfigWithUser(user, cf.kubeConfigOptions...)(cluster)
			if err != nil {
				return nil, fmt.Errorf("error creating client for cluster: %w", err)
			}

			break
		}
	}

	if config == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating discovery client for config: %w", err)
	}

	return dc, nil
}

func (cf *clientsFactory) GetServerClient(ctx context.Context) (Client, error) {
	pool := cf.newClustersPool(cf.scheme)
	errChan := make(chan error, len(cf.clusters.Get()))

	var wg sync.WaitGroup

	for _, cluster := range cf.clusters.Get() {
		wg.Add(1)

		go func(cluster Cluster, pool ClientsPool, errChan chan error) {
			defer wg.Done()

			if err := pool.Add(ClientConfigAsServer(cf.kubeConfigOptions...), cluster); err != nil {
				errChan <- &ClientError{ClusterName: cluster.Name, Err: fmt.Errorf("failed adding cluster client to pool: %w", err)}
			}
		}(cluster, pool, errChan)
	}

	wg.Wait()
	close(errChan)

	var result *multierror.Error

	for err := range errChan {
		result = multierror.Append(result, err)
	}

	return NewClient(pool, cf.clustersNamespaces.namespaces), result.ErrorOrNil()
}

func (cf *clientsFactory) UpdateUserNamespaces(ctx context.Context, user *auth.UserPrincipal) {
	wg := sync.WaitGroup{}

	for _, cluster := range cf.clusters.Get() {
		wg.Add(1)

		go func(cluster Cluster) {
			defer wg.Done()

			clusterNs := cf.clustersNamespaces.Get(cluster.Name)

			cfg, err := ClientConfigWithUser(user, cf.kubeConfigOptions...)(cluster)
			if err != nil {
				cf.log.Error(err, "failed creating client config", "cluster", cluster.Name, "user", user.ID)
				return
			}

			filteredNs, err := cf.nsChecker.FilterAccessibleNamespaces(ctx, cfg, clusterNs)
			if err != nil {
				cf.log.Error(err, "failed filtering namespaces", "cluster", cluster.Name, "user", user.ID)
				return
			}

			cf.usersNamespaces.Set(user, cluster.Name, filteredNs)
		}(cluster)
	}

	wg.Wait()
}

func (cf *clientsFactory) GetUserNamespaces(user *auth.UserPrincipal) map[string][]v1.Namespace {
	return cf.usersNamespaces.GetAll(user, cf.clusters.Get())
}

func (cf *clientsFactory) userNsList(ctx context.Context, user *auth.UserPrincipal) map[string][]v1.Namespace {
	userNamespaces := cf.GetUserNamespaces(user)
	if len(userNamespaces) > 0 {
		return userNamespaces
	}

	cf.UpdateUserNamespaces(ctx, user)

	return cf.GetUserNamespaces(user)
}

func ApplyKubeConfigOptions(config *rest.Config, options ...KubeConfigOption) (*rest.Config, error) {
	for _, o := range options {
		_, err := o(config)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// restConfigFromCluster creates a generic rest.Config for a given cluster.
// You should not call this directly, but rather via
// ClientConfigAsServer or ClientConfigWithUser
func restConfigFromCluster(cluster Cluster) *rest.Config {
	return &rest.Config{
		Host:            cluster.Server,
		TLSClientConfig: cluster.TLSConfig,
		QPS:             ClientQPS,
		Burst:           ClientBurst,
		Timeout:         kubeClientTimeout,
		Dial: (&net.Dialer{
			Timeout: kubeClientDialTimeout,
			// KeepAlive is default to 30s within client-go.
			KeepAlive: kubeClientDialKeepAlive,
		}).DialContext,
	}
}

func WithFlowControl(config *rest.Config) (*rest.Config, error) {
	// flowcontrol.IsEnabled makes a request to the K8s API of the cluster stored in the config.
	// It does a HEAD request to /livez/ping which uses the config.Dial timeout. We can use this
	// function to error early rather than wait to call client.New.
	enabled, err := flowcontrol.IsEnabled(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("error querying cluster for flowcontrol config: %w", err)
	}

	if enabled {
		// Enabled & negative QPS and Burst indicate that the client would use the rate limit set by the server.
		// Ref: https://github.com/kubernetes/kubernetes/blob/v1.24.0/staging/src/k8s.io/client-go/rest/config.go#L354-L364
		config.QPS = -1
		config.Burst = -1

		return config, nil
	}

	config.QPS = ClientQPS
	config.Burst = ClientBurst

	return config, nil
}

// clientConfigAsServer returns a *rest.Config for a given cluster
// as the server service acconut
func ClientConfigAsServer(options ...KubeConfigOption) ClusterClientConfigFunc {
	return func(cluster Cluster) (*rest.Config, error) {
		config := restConfigFromCluster(cluster)

		config.BearerToken = cluster.BearerToken

		return ApplyKubeConfigOptions(config, options...)
	}
}

// ClientConfigWithUser returns a function that returns a *rest.Config with the relevant
// user authentication details pre-defined for a given cluster.
func ClientConfigWithUser(user *auth.UserPrincipal, options ...KubeConfigOption) ClusterClientConfigFunc {
	return func(cluster Cluster) (*rest.Config, error) {
		config := restConfigFromCluster(cluster)

		if !user.Valid() {
			return nil, fmt.Errorf("No user ID or Token found in UserPrincipal.")
		} else if tok := user.Token(); tok != "" {
			config.BearerToken = tok
		} else {
			config.BearerToken = cluster.BearerToken
			config.Impersonate = rest.ImpersonationConfig{
				UserName: user.ID,
				Groups:   user.Groups,
			}
		}

		return ApplyKubeConfigOptions(config, options...)
	}
}
