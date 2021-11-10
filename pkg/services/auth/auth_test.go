package auth

import (
	"bytes"
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/weave-gitops/pkg/flux"
	"github.com/weaveworks/weave-gitops/pkg/gitproviders"
	"github.com/weaveworks/weave-gitops/pkg/gitproviders/gitprovidersfakes"
	"github.com/weaveworks/weave-gitops/pkg/logger"
	"github.com/weaveworks/weave-gitops/pkg/osys"
	"github.com/weaveworks/weave-gitops/pkg/runner"
	"github.com/weaveworks/weave-gitops/pkg/services/automation"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

type actualFluxRunner struct {
	runner.Runner
}

func (r *actualFluxRunner) Run(command string, args ...string) ([]byte, error) {
	cmd := "../../flux/bin/flux"

	return r.Runner.Run(cmd, args...)
}

var _ = Describe("auth", func() {
	var namespace *corev1.Namespace
	testClustername := "test-cluster"
	repoUrlString := "ssh://git@github.com/my-org/my-repo.git"
	repoUrl, err := gitproviders.NewRepoURL(repoUrlString)
	Expect(err).NotTo(HaveOccurred())
	BeforeEach(func() {
		namespace = &corev1.Namespace{}
		namespace.Name = "kube-test-" + rand.String(5)

		Expect(k8sClient.Create(context.Background(), namespace)).To(Succeed())
	})
	Describe("AuthService", func() {
		var (
			ctx        context.Context
			secretName automation.GeneratedSecretName
			gp         gitprovidersfakes.FakeGitProvider
			osysClient osys.Osys
			as         AuthService
			fluxClient flux.Flux
		)
		BeforeEach(func() {
			ctx = context.Background()
			secretName = automation.CreateRepoSecretName(testClustername, repoUrl)
			Expect(err).NotTo(HaveOccurred())
			osysClient = osys.New()
			gp = gitprovidersfakes.FakeGitProvider{}
			fluxClient = flux.New(osysClient, &actualFluxRunner{Runner: &runner.CLIRunner{}})

			as = &authSvc{
				logger:      logger.NewCLILogger(bytes.NewBuffer([]byte{})), //Stay silent in tests.
				fluxClient:  fluxClient,
				k8sClient:   k8sClient,
				gitProvider: &gp,
			}
		})
		It("create and stores a deploy key if none exists", func() {
			_, err := as.CreateGitClient(ctx, repoUrl, testClustername, namespace.Name)
			Expect(err).NotTo(HaveOccurred())
			sn := SecretName{Name: secretName, Namespace: namespace.Name}
			secret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, sn.NamespacedName(), secret)).To(Succeed())

			Expect(secret.StringData["identity"]).NotTo(BeNil())
			Expect(secret.StringData["identity.pub"]).NotTo(BeNil())
		})
		It("uses an existing deploy key when present", func() {
			gp.DeployKeyExistsReturns(true, nil)
			sn := SecretName{Name: secretName, Namespace: namespace.Name}
			// using `generateDeployKey` as a helper for the test setup.
			_, secret, err := (&authSvc{fluxClient: fluxClient}).generateDeployKey(testClustername, sn, repoUrl)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			_, err = as.CreateGitClient(ctx, repoUrl, testClustername, namespace.Name)
			Expect(err).NotTo(HaveOccurred())
			// We should NOT have uploaded anything since the key already exists
			Expect(gp.UploadDeployKeyCallCount()).To(Equal(0))
		})
		It("handles the case where a deploy key exists on the provider, but not the cluster", func() {
			gp.DeployKeyExistsReturns(true, nil)
			sn := SecretName{Name: secretName, Namespace: namespace.Name}

			_, err = as.CreateGitClient(ctx, repoUrl, testClustername, namespace.Name)
			Expect(err).NotTo(HaveOccurred())

			newSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, sn.NamespacedName(), newSecret)).To(Succeed())
			Expect(gp.UploadDeployKeyCallCount()).To(Equal(1))
		})
	})
})
