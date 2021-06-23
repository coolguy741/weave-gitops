package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/sirupsen/logrus"
	pb "github.com/weaveworks/weave-gitops/pkg/api/applications"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/runner"
	"github.com/weaveworks/weave-gitops/pkg/server"
)

var log = logrus.New()

func main() {
	mux := http.NewServeMux()

	mux.Handle("/health/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))

		if err != nil {
			log.Errorf("error writing health check: %s", err)
		}
	}))

	assetFS := getAssets()
	assetHandler := http.FileServer(http.FS(assetFS))
	redirector := createRedirector(assetFS, log)

	gMux := runtime.NewServeMux()
	mux.Handle("/v1/", gMux)

	kubeClient := kube.New(&runner.CLIRunner{})

	if err := pb.RegisterApplicationsHandlerServer(context.Background(), gMux, server.NewApplicationsServer(kubeClient)); err != nil {
		log.Fatalf("could not register application: %s", err)
	}

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Assume anything with a file extension in the name is a static asset.
		extension := filepath.Ext(req.URL.Path)
		// We use the golang http.FileServer for static file requests.
		// This will return a 404 on normal page requests, ie /some-page.
		// Redirect all non-file requests to index.html, where the JS routing will take over.
		if extension == "" {
			redirector(w, req)
			return
		}
		assetHandler.ServeHTTP(w, req)
	}))

	port := ":9001"

	log.Infof("Serving on port %s", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Error(err, "server exited")
		os.Exit(1)
	}
}

//go:embed dist/*
var static embed.FS

func getAssets() fs.FS {
	f, err := fs.Sub(static, "dist")

	if err != nil {
		panic(err)
	}
	return f
}

// A redirector ensures that index.html always gets served.
// The JS router will take care of actual navigation once the index.html page lands.
func createRedirector(fsys fs.FS, log logrus.FieldLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		indexPage, err := fsys.Open("index.html")

		if err != nil {
			log.Error(err, "could not open index.html page")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		stat, err := indexPage.Stat()
		if err != nil {
			log.Error(err, "could not get index.html stat")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bt := make([]byte, stat.Size())
		_, err = indexPage.Read(bt)

		if err != nil {
			log.Error(err, "could not read index.html")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = w.Write(bt)

		if err != nil {
			log.Error(err, "error writing index.html")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
