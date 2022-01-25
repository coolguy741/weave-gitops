package watcher

import (
	"io/ioutil"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/weaveworks/weave-gitops/pkg/helm"
	//+kubebuilder:scaffold:imports
	"github.com/weaveworks/weave-gitops/pkg/helm/watcher/cache"
	"github.com/weaveworks/weave-gitops/pkg/helm/watcher/controller"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

type Options struct {
	KubeClient         client.Client
	Cache              cache.Cache
	MetricsBindAddress string
	HealthzBindAddress string
	WatcherPort        int
}

type Watcher struct {
	cache              cache.Cache
	repoManager        helm.HelmRepoManager
	metricsBindAddress string
	healthzBindAddress string
	watcherPort        int
}

func NewWatcher(opts Options) (*Watcher, error) {
	tempDir, err := ioutil.TempDir("", "profile_cache")
	if err != nil {
		return nil, err
	}

	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}

	if err := sourcev1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	return &Watcher{
		cache:              opts.Cache,
		repoManager:        helm.NewRepoManager(opts.KubeClient, tempDir),
		healthzBindAddress: opts.HealthzBindAddress,
		metricsBindAddress: opts.MetricsBindAddress,
		watcherPort:        opts.WatcherPort,
	}, nil
}

func (w *Watcher) StartWatcher() error {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{
		Development: false,
	})))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     w.metricsBindAddress,
		HealthProbeBindAddress: w.healthzBindAddress,
		Port:                   w.watcherPort,
		Logger:                 ctrl.Log,
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		return err
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	if err = (&controller.HelmWatcherReconciler{
		Client:      mgr.GetClient(),
		Cache:       w.cache,
		RepoManager: w.repoManager,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HelmWatcherReconciler")
		return err
	}

	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}
