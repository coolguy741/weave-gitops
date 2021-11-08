//go:build !unittest
// +build !unittest

package server_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/fluxcd/go-git-providers/github"
	"github.com/fluxcd/go-git-providers/gitprovider"
	"github.com/go-logr/zapr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pb "github.com/weaveworks/weave-gitops/pkg/api/applications"
	"github.com/weaveworks/weave-gitops/pkg/flux"
	"github.com/weaveworks/weave-gitops/pkg/logger"
	"github.com/weaveworks/weave-gitops/pkg/osys"
	"github.com/weaveworks/weave-gitops/pkg/runner"
	"github.com/weaveworks/weave-gitops/pkg/server"
	"github.com/weaveworks/weave-gitops/pkg/services"
	"github.com/weaveworks/weave-gitops/pkg/services/auth"
	"github.com/weaveworks/weave-gitops/pkg/testutils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	corev1 "k8s.io/api/core/v1"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener
var env *testutils.K8sTestEnv
var gp gitprovider.Client
var org = "weaveworks-gitops-test"
var conn *grpc.ClientConn
var s *grpc.Server
var err error
var clusterName = "test-cluster"

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

var stop func()

func TestServerIntegration(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Server Integration")
}

var _ = BeforeSuite(func() {
	ctx := context.Background()
	env, err = testutils.StartK8sTestEnvironment([]string{
		"../../../manifests/crds",
		"../../../tools/testcrds",
	})
	Expect(err).NotTo(HaveOccurred())

	fluxNs := &corev1.Namespace{}
	fluxNs.Name = "flux-system"

	Expect(env.Client.Create(ctx, fluxNs)).To(Succeed())

	stop = env.Stop
	fluxClient := flux.New(osys.New(), &runner.CLIRunner{})
	fluxClient.SetupBin()

	gp, err = github.NewClient(
		gitprovider.WithDestructiveAPICalls(true),
		gitprovider.WithOAuth2Token(os.Getenv("GITHUB_TOKEN")),
	)
	Expect(err).NotTo(HaveOccurred())

	factory := services.NewServerFactory(fluxClient, logger.NewApiLogger(zap.NewNop()), env.Rest, clusterName)
	Expect(err).NotTo(HaveOccurred())

	cfg := &server.ApplicationsConfig{
		Factory:          factory,
		Logger:           zapr.NewLogger(zap.NewNop()),
		JwtClient:        auth.NewJwtClient("somekey"),
		GithubAuthClient: auth.NewGithubAuthProvider(http.DefaultClient),
		KubeClient:       env.Client,
	}

	s = grpc.NewServer()
	apps := server.NewApplicationsServer(cfg)
	pb.RegisterApplicationsServer(s, apps)

	go func() {
		if err := s.Serve(lis); err != nil {
			fmt.Println(err.Error())
		}
	}()

	lis = bufconn.Listen(bufSize)

	conn, err = grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	stop()
	conn.Close()
	s.Stop()
})
