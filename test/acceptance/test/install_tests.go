/**
* All tests related to 'gitops install' will go into this file
 */

package acceptance

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	wego "github.com/weaveworks/weave-gitops/api/v1alpha1"
	"github.com/weaveworks/weave-gitops/manifests"
	"github.com/weaveworks/weave-gitops/pkg/kube"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Weave GitOps Install Tests", func() {

	var sessionOutput *gexec.Session

	BeforeEach(func() {

		By("Given I have a gitops binary installed on my local machine", func() {
			Expect(FileExists(gitopsBinaryPath)).To(BeTrue())
		})
	})

	It("Validate that gitops displays help text for 'install' command", func() {

		By("When I run the command 'gitops install -h'", func() {
			sessionOutput = runCommandAndReturnSessionOutput(gitopsBinaryPath + " install -h")
		})

		By("Then I should see gitops help text displayed for 'install' command", func() {
			helpTest := fmt.Sprintf(`The install command deploys GitOps in the specified namespace,
adds a cluster entry to the GitOps repo, and persists the GitOps runtime into the
repo. If a previous version is installed, then an in-place upgrade will be performed.

Usage:
  gitops install [flags]

Examples:
  # Install GitOps in the %s namespace
  gitops install --config-repo=ssh://git@github.com/me/mygitopsrepo.git

Flags:
      --auto-merge           If set, 'gitops install' will automatically update the default branch for the configuration repository
      --config-repo string   URL of external repository that will hold automation manifests
      --dry-run              Outputs all the manifests that would be installed
  -h, --help                 help for install

Global Flags:
  -e, --endpoint string    The Weave GitOps Enterprise HTTP API endpoint
      --namespace string   The namespace scope for this operation (default "%s")
  -v, --verbose            Enable verbose output`, wego.DefaultNamespace, wego.DefaultNamespace)
			helpTest = regexp.QuoteMeta(helpTest)
			Eventually(sessionOutput).Should(gbytes.Say(helpTest))
		})
	})

	It("Validate that gitops displays help text for 'uninstall' command", func() {

		By("When I run the command 'gitops uninstall -h'", func() {
			sessionOutput = runCommandAndReturnSessionOutput(gitopsBinaryPath + " uninstall -h")
		})

		By("Then I should see gitops help text displayed for 'uninstall' command", func() {
			Eventually(string(sessionOutput.Wait().Out.Contents())).Should(MatchRegexp(
				fmt.Sprintf(`The uninstall command removes GitOps components from the cluster.\n*Usage:\n\s*gitops uninstall \[flags]\n*Examples:\n\s*# Uninstall GitOps from the %s namespace\n\s*gitops uninstall\n*Flags:\n\s*--dry-run\s*Outputs all the manifests that would be uninstalled\n\s*--force\s*If set, 'gitops uninstall' will not ask for confirmation\n\s*-h, --help\s*help for uninstall\n*Global Flags:\n\s*-e, --endpoint string\s*The Weave GitOps Enterprise HTTP API endpoint\n\s*--namespace string\s*The namespace scope for this operation \(default "%s"\)\n\s*-v, --verbose\s*Enable verbose output`, wego.DefaultNamespace, wego.DefaultNamespace)))
		})
	})

	It("Verify that gitops quits if flux-system namespace is present", func() {
		var errOutput string
		namespace := "flux-system"

		defer deleteNamespace(namespace)

		By("And I have a brand new cluster", func() {
			_, _, err := ResetOrCreateCluster(WEGO_DEFAULT_NAMESPACE, true)
			Expect(err).ShouldNot(HaveOccurred())
		})

		By("When I create a '"+namespace+"' namespace", func() {
			namespaceCreatedMsg := runCommandAndReturnSessionOutput("kubectl create ns " + namespace)
			Eventually(namespaceCreatedMsg).Should(gbytes.Say("namespace/" + namespace + " created"))
		})

		By("And I run 'gitops install' command", func() {
			_, errOutput = runCommandAndReturnStringOutput(gitopsBinaryPath + " install --config-repo=ssh://git@" + gitProviderName + ".com/user/repo.git")
		})

		By("Then I should see a quitting message", func() {
			Eventually(errOutput).Should(MatchRegexp(
				`Error: Weave GitOps does not yet support installation onto a cluster that is using Flux.\nPlease uninstall flux before proceeding:\n\s*. flux uninstall`))
		})
	})

	It("Verify that gitops can install & uninstall gitops components under a user-specified namespace", func() {

		namespace := "test-namespace"

		By("And I have a brand new cluster", func() {
			_, _, err := ResetOrCreateCluster(namespace, true)
			Expect(err).ShouldNot(HaveOccurred())
		})

		private := true
		tip := generateTestInputs()
		appRepoRemoteURL := "git@" + gitProviderName + ".com:" + gitOrg + "/" + tip.appRepoName + ".git"

		defer deleteRepo(tip.appRepoName, gitProvider, gitOrg)

		By("And application repo does not already exist", func() {
			deleteRepo(tip.appRepoName, gitProvider, gitOrg)
		})

		_ = initAndCreateEmptyRepo(tip.appRepoName, gitProvider, private, gitOrg)

		installAndVerifyWego(namespace, appRepoRemoteURL)

		By("When I run 'gitops uninstall' command without force flag it asks for confirmation", func() {
			cmd := fmt.Sprintf("%s uninstall --namespace %s", gitopsBinaryPath, namespace)
			outputStream := gbytes.NewBuffer()
			inputUser := bytes.NewBuffer([]byte("y\n"))

			c := exec.Command("sh", "-c", cmd)
			c.Stdout = outputStream
			c.Stdin = inputUser
			c.Stderr = os.Stderr
			err := c.Start()
			Expect(err).ShouldNot(HaveOccurred())

			Eventually(outputStream).Should(gbytes.Say(`Uninstall will remove all your Applications and any related cluster resources\. Are you sure you want to uninstall\? \[y\/N\]`))

			err = c.Wait()
			Expect(err).ShouldNot(HaveOccurred())
		})

		_ = waitForNamespaceToTerminate(namespace, NAMESPACE_TERMINATE_TIMEOUT)

		By("Then I should not see any gitops components", func() {
			_, errOutput := runCommandAndReturnStringOutput("kubectl get ns " + namespace)
			Eventually(errOutput).Should(ContainSubstring(`Error from server (NotFound): namespaces "` + namespace + `" not found`))
		})
	})

	It("@skipOnNightly Verify that gitops can install(via pull requests) & uninstall(via auto-merge) gitops components to multiple clusters under a user-specified namespace", func() {
		namespace := "test-namespace"

		_, cluster1Context, err := ResetOrCreateCluster(namespace, true)
		Expect(err).ShouldNot(HaveOccurred())

		private := true
		tip := generateTestInputs()
		appRepoRemoteURL := "git@github.com:" + githubOrg + "/" + tip.appRepoName + ".git"

		defer deleteRepo(tip.appRepoName, gitproviders.GitProviderGitHub, githubOrg)

		By("And application repo does not already exist", func() {
			deleteRepo(tip.appRepoName, gitproviders.GitProviderGitHub, githubOrg)
		})

		repoAbsolutePath := initAndCreateEmptyRepo(tip.appRepoName, gitproviders.GitProviderGitHub, private, githubOrg)

		installAndVerifyWegoViaPullRequest(namespace, appRepoRemoteURL, repoAbsolutePath)

		cluster2Name, cluster2Context, err := ResetOrCreateClusterWithName(namespace, false, "", true)
		Expect(err).ShouldNot(HaveOccurred())

		defer func() {
			selectCluster(cluster2Context)
			deleteCluster(cluster2Name)
			selectCluster(cluster1Context)
		}()

		selectCluster(cluster2Context)
		installAndVerifyWegoViaPullRequest(namespace, appRepoRemoteURL, repoAbsolutePath)

		selectCluster(cluster1Context)
		cmd := fmt.Sprintf("%s uninstall --namespace %s --force", gitopsBinaryPath, namespace)
		c := exec.Command("sh", "-c", cmd)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		err = c.Start()

		Expect(err).ShouldNot(HaveOccurred())

		err = c.Wait()

		Expect(err).ShouldNot(HaveOccurred())

		_ = waitForNamespaceToTerminate(namespace, NAMESPACE_TERMINATE_TIMEOUT)

		By("Then I should not see any gitops components", func() {
			_, errOutput := runCommandAndReturnStringOutput("kubectl get ns " + namespace)
			Eventually(errOutput).Should(ContainSubstring(`Error from server (NotFound): namespaces "` + namespace + `" not found`))
		})

		selectCluster(cluster2Context)

		cmd = fmt.Sprintf("%s uninstall --namespace %s --force", gitopsBinaryPath, namespace)
		c = exec.Command("sh", "-c", cmd)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		err = c.Start()

		Expect(err).ShouldNot(HaveOccurred())

		err = c.Wait()
		Expect(err).ShouldNot(HaveOccurred())

		_ = waitForNamespaceToTerminate(namespace, NAMESPACE_TERMINATE_TIMEOUT)

		By("Then I should not see any gitops components", func() {
			Expect(waitForNamespaceToTerminate(namespace, NAMESPACE_TERMINATE_TIMEOUT)).ShouldNot(HaveOccurred())
		})
	})

	It("Verify that gitops can uninstall flux if gitops was not fully installed", func() {

		namespace := "test-namespace"

		By("And I have a brand new cluster", func() {
			_, _, err := ResetOrCreateCluster(namespace, true)
			Expect(err).ShouldNot(HaveOccurred())
		})

		private := true
		tip := generateTestInputs()
		appRepoRemoteURL := "ssh://git@" + gitProviderName + ".com/" + gitOrg + "/" + tip.appRepoName + ".git"

		defer deleteRepo(tip.appRepoName, gitProvider, gitOrg)

		By("And application repo does not already exist", func() {
			deleteRepo(tip.appRepoName, gitProvider, gitOrg)
		})

		_ = initAndCreateEmptyRepo(tip.appRepoName, gitProvider, private, gitOrg)

		installAndVerifyWego(namespace, appRepoRemoteURL)

		ctx := context.Background()

		kubeClient, _, kubeErr := kube.NewKubeHTTPClient()
		Expect(kubeErr).ShouldNot(HaveOccurred())

		crdErr := kubeClient.Delete(ctx, manifests.AppCRD)
		Expect(crdErr).ShouldNot(HaveOccurred())

		By("When I run 'gitops uninstall' command", func() {
			runErr := runCommandPassThrough([]string{}, "sh", "-c", fmt.Sprintf("%s uninstall --force --namespace %s", gitopsBinaryPath, namespace))
			Expect(runErr).ShouldNot(HaveOccurred())
		})

		_ = waitForNamespaceToTerminate(namespace, NAMESPACE_TERMINATE_TIMEOUT)

		By("Then I should not see any gitops components", func() {
			_, errOutput := runCommandAndReturnStringOutput("kubectl get ns " + namespace)
			Eventually(errOutput).Should(ContainSubstring(`Error from server (NotFound): namespaces "` + namespace + `" not found`))
		})
	})

	It("Verify that gitops can: install gitops components, uninstall gitops components, and work in dry-run mode", func() {

		var installDryRunOutput string
		var uninstallDryRunOutput string

		By("And I have a brand new cluster", func() {
			_, _, err := ResetOrCreateCluster(WEGO_DEFAULT_NAMESPACE, true)
			Expect(err).ShouldNot(HaveOccurred())
		})

		private := true
		tip := generateTestInputs()
		appRepoRemoteURL := "ssh://git@" + gitProviderName + ".com/" + gitOrg + "/" + tip.appRepoName + ".git"

		defer deleteRepo(tip.appRepoName, gitProvider, gitOrg)

		By("And application repo does not already exist", func() {
			deleteRepo(tip.appRepoName, gitProvider, gitOrg)
		})

		_ = initAndCreateEmptyRepo(tip.appRepoName, gitProvider, private, gitOrg)

		By("When I try to install gitops in dry-run mode", func() {
			var errOnInstall string
			installDryRunOutput, errOnInstall = runCommandAndReturnStringOutput(gitopsBinaryPath + fmt.Sprintf(" install --dry-run --config-repo=%s", appRepoRemoteURL))
			Expect(errOnInstall).To(BeEmpty())
		})

		By("Then I should see install dry-run output in the console", func() {
			Expect(installDryRunOutput).Should(ContainSubstring("# Flux Version: "))
			Expect(installDryRunOutput).Should(ContainSubstring("# Components: source-controller,kustomize-controller,helm-controller,notification-controller,image-reflector-controller,image-automation-controller"))
			Expect(installDryRunOutput).Should(ContainSubstring("name: " + WEGO_DEFAULT_NAMESPACE))
		})

		By("And gitops components should be absent from the cluster", func() {
			_, err := runCommandAndReturnStringOutput("kubectl get ns " + WEGO_DEFAULT_NAMESPACE)
			Eventually(err).Should(ContainSubstring(`Error from server (NotFound): namespaces "` + WEGO_DEFAULT_NAMESPACE + `" not found`))
		})

		installAndVerifyWego(WEGO_DEFAULT_NAMESPACE, appRepoRemoteURL)

		By("When I try to uninstall gitops in dry-run mode", func() {
			uninstallDryRunOutput, _ = runCommandAndReturnStringOutput(gitopsBinaryPath + " uninstall --force --dry-run")
		})

		By("Then I should see uninstall dry-run output in the console", func() {
			Eventually(uninstallDryRunOutput).Should(ContainSubstring("► deleting components in " + WEGO_DEFAULT_NAMESPACE + " namespace"))
			Eventually(uninstallDryRunOutput).Should(ContainSubstring(fmt.Sprintf("✔ Deployment/%s/helm-controller deleted (dry run)", wego.DefaultNamespace)))
			Eventually(uninstallDryRunOutput).Should(ContainSubstring(fmt.Sprintf("✔ Deployment/%s/image-automation-controller deleted (dry run)", wego.DefaultNamespace)))
			Eventually(uninstallDryRunOutput).Should(ContainSubstring(fmt.Sprintf("✔ Deployment/%s/image-reflector-controller deleted (dry run)", wego.DefaultNamespace)))
			Eventually(uninstallDryRunOutput).Should(ContainSubstring(fmt.Sprintf("✔ Deployment/%s/kustomize-controller deleted (dry run)", wego.DefaultNamespace)))
			Eventually(uninstallDryRunOutput).Should(ContainSubstring(fmt.Sprintf("✔ Deployment/%s/notification-controller deleted (dry run)", wego.DefaultNamespace)))
			Eventually(uninstallDryRunOutput).Should(ContainSubstring(fmt.Sprintf("✔ Deployment/%s/source-controller deleted (dry run)", wego.DefaultNamespace)))
			Eventually(uninstallDryRunOutput).Should(ContainSubstring(fmt.Sprintf("✔ Namespace/%s deleted (dry run)", wego.DefaultNamespace)))
			Eventually(uninstallDryRunOutput).Should(ContainSubstring("✔ uninstall finished"))
		})

		By("And gitops components should be present in the cluster", func() {
			VerifyControllersInCluster(WEGO_DEFAULT_NAMESPACE)
		})

		By("When I run 'gitops uninstall' command", func() {
			_ = runCommandPassThrough([]string{}, "sh", "-c", fmt.Sprintf("%s uninstall --force --namespace %s", gitopsBinaryPath, WEGO_DEFAULT_NAMESPACE))
		})

		_ = waitForNamespaceToTerminate(WEGO_DEFAULT_NAMESPACE, NAMESPACE_TERMINATE_TIMEOUT)

		By("Then I should not see any gitops components", func() {
			_, errOutput := runCommandAndReturnStringOutput("kubectl get ns " + WEGO_DEFAULT_NAMESPACE)
			Eventually(errOutput).Should(ContainSubstring(`Error from server (NotFound): namespaces "` + WEGO_DEFAULT_NAMESPACE + `" not found`))
		})
	})

	It("Verify wego app is deployed", func() {
		namespace := "wego-system"

		By("And I have a brand new cluster", func() {
			_, _, err := ResetOrCreateCluster(namespace, true)
			Expect(err).ShouldNot(HaveOccurred())
		})

		private := true
		tip := generateTestInputs()
		appRepoRemoteURL := "ssh://git@" + gitProviderName + ".com/" + gitOrg + "/" + tip.appRepoName + ".git"

		defer deleteRepo(tip.appRepoName, gitProvider, gitOrg)

		By("And application repo does not already exist", func() {
			deleteRepo(tip.appRepoName, gitProvider, gitOrg)
		})

		_ = initAndCreateEmptyRepo(tip.appRepoName, gitProvider, private, gitOrg)

		installAndVerifyWego(namespace, appRepoRemoteURL)

		By("And the wego-app is up and running", func() {
			command := exec.Command("sh", "-c", fmt.Sprintf("kubectl wait --for=condition=Ready --timeout=60s -n %s --all pods --selector='app=wego-app'", namespace))
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session, INSTALL_PODS_READY_TIMEOUT).Should(gexec.Exit())
		})

		_ = waitForNamespaceToTerminate(namespace, NAMESPACE_TERMINATE_TIMEOUT)
	})
})
