/**
* All common util functions and golbal constants will go here.
**/
package acceptance

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	log "github.com/sirupsen/logrus"
)

const EVENTUALLY_DEFAULT_TIME_OUT time.Duration = 60 * time.Second
const TIMEOUT_TWO_MINUTES time.Duration = 120 * time.Second
const INSTALL_RESET_TIMEOUT time.Duration = 300 * time.Second
const NAMESPACE_TERMINATE_TIMEOUT time.Duration = 600 * time.Second
const INSTALL_PODS_READY_TIMEOUT time.Duration = 180 * time.Second
const WEGO_DEFAULT_NAMESPACE = "wego-system"

var DEFAULT_SSH_KEY_PATH string
var GITHUB_ORG string
var WEGO_BIN_PATH string

type TestInputs struct {
	appRepoName         string
	appManifestFilePath string
	workloadName        string
	workloadNamespace   string
}

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func RandString(length int) string {
	return StringWithCharset(length, charset)
}

func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func ShowWegoControllerLogs(ns string) {
	controllers := []string{"helm", "kustomize", "notification", "source"}

	for _, c := range controllers {
		label := c + "-controller"
		log.Infof("Logs for controller: %s", label)
		_ = runCommandPassThrough([]string{}, "kubectl", "logs", "-l", "app="+label, "-n", ns)
	}
}

// showItems displays the current set of a specified object type in tabular format
func ShowItems(itemType string) error {
	if itemType != "" {
		return runCommandPassThrough([]string{}, "kubectl", "get", itemType, "--all-namespaces", "-o", "wide")
	}
	return runCommandPassThrough([]string{}, "kubectl", "get", "all", "--all-namespaces", "-o", "wide")
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func waitForNamespaceToTerminate(namespace string, timeout time.Duration) error {
	//check if the namespace exist before cleaning up
	pollInterval := 5
	if timeout < 5*time.Second {
		timeout = 5 * time.Second
	}
	timeoutInSeconds := int(timeout.Seconds())

	err := runCommandPassThrough([]string{}, "sh", "-c", fmt.Sprintf("kubectl get ns %s", namespace))
	if err != nil {
		log.Infof("Namespace %s doesn't exist, nothing to clean, Skipping...", namespace)
		return nil
	}

	for i := pollInterval; i < timeoutInSeconds; i += pollInterval {
		log.Infof("Waiting for namespace: %s to terminate : %d second(s) passed of %d seconds timeout", namespace, i, timeoutInSeconds)
		out, _ := runCommandAndReturnOutput(fmt.Sprintf("kubectl get ns %s --ignore-not-found=true | grep -i terminating", namespace))
		out = strings.TrimSpace(out)
		if out == "" {
			return nil
		}
		if i > timeoutInSeconds/2 && i%10 == 0 {
			//Patch the finalizer
			log.Infof("Patch the finalizer to unstuck the terminating namespace %s", namespace)
			_ = runCommandPassThrough([]string{}, "sh", "-c", fmt.Sprintf("kubectl patch ns %s -p '{\"metadata\":{\"finalizers\":[]}}' --type=merge", namespace))
		}
		time.Sleep(time.Duration(pollInterval) * time.Second)
	}
	return fmt.Errorf("Error: Failed to terminate the namespace %s", namespace)
}

func uninstallWegoRuntime(namespace string) {
	log.Infof("About to delete WeGO runtime from namespace: %s", namespace)
	err := runCommandPassThrough([]string{}, "sh", "-c", fmt.Sprintf("%s flux uninstall --namespace %s --silent", WEGO_BIN_PATH, namespace))
	if err != nil {
		log.Infof("Failed to uninstall the wego runtime %s", namespace)
	}

	err = runCommandPassThrough([]string{}, "sh", "-c", "kubectl delete crd apps.wego.weave.works")
	if err != nil {
		log.Infof("Failed to delete crd apps.wego.weave.works")
	}
	Expect(waitForNamespaceToTerminate(namespace, NAMESPACE_TERMINATE_TIMEOUT)).To(Succeed())
}

func ResetOrCreateCluster(namespace string, deleteWegoRuntime bool) (string, error) {
	supportedProviders := []string{"kind", "kubectl"}
	supportedK8SVersions := []string{"1.19.1", "1.20.2", "1.21.1"}
	clusterName := ""

	provider, found := os.LookupEnv("CLUSTER_PROVIDER")
	if !found {
		provider = "kind"
	}

	k8sVersion, found := os.LookupEnv("K8S_VERSION")
	if !found {
		k8sVersion = "1.20.2"
	}

	if !contains(supportedProviders, provider) {
		log.Errorf("Cluster provider %s is not supported for testing", provider)
		return clusterName, errors.New("Unsupported provider")
	}

	if !contains(supportedK8SVersions, k8sVersion) {
		log.Errorf("Kubernetes version %s is not supported for testing", k8sVersion)
		return clusterName, errors.New("Unsupported kubernetes version")
	}

	//For kubectl, point to a valid cluster, we will try to reset the namespace only
	if namespace != "" && provider == "kubectl" {
		err := runCommandPassThrough([]string{}, "./scripts/reset-wego.sh", namespace)
		if err != nil {
			log.Infof("Failed to reset the wego runtime in namespace %s", namespace)
		}

		if deleteWegoRuntime {
			uninstallWegoRuntime(namespace)
		}

		out, err := exec.Command("kubectl", "config", "current-context").Output()
		Expect(err).ShouldNot(HaveOccurred())
		clusterName = string(bytes.TrimSuffix(out, []byte("\n")))
	}

	if provider == "kind" {
		clusterName = provider + "-" + RandString(6)
		log.Infof("Creating a kind cluster %s", clusterName)
		err := runCommandPassThrough([]string{}, "./scripts/kind-cluster.sh", clusterName, "kindest/node:v"+k8sVersion)
		if err != nil {
			log.Infof("Failed to create kind cluster")
			log.Fatal(err)
			return clusterName, err
		}
	}

	log.Info("Wait for the cluster to be ready")
	err := runCommandPassThrough([]string{}, "kubectl", "wait", "--for=condition=Ready", "--timeout=300s", "-n", "kube-system", "--all", "pods")
	if err != nil {
		log.Infof("Cluster system pods are not ready after waiting for 5 minutes, This can cause tests failures.")
		return clusterName, err
	}

	return clusterName, nil
}

func waitForResource(resourceType string, resourceName string, namespace string, timeout time.Duration) error {
	pollInterval := 5
	if timeout < 5*time.Second {
		timeout = 5 * time.Second
	}

	timeoutInSeconds := int(timeout.Seconds())
	for i := pollInterval; i < timeoutInSeconds; i += pollInterval {
		log.Infof("Waiting for %s in namespace: %s... : %d second(s) passed of %d seconds timeout", resourceType+"/"+resourceName, namespace, i, timeoutInSeconds)
		err := runCommandPassThroughWithoutOutput([]string{}, "sh", "-c", fmt.Sprintf("kubectl get %s %s -n %s", resourceType, resourceName, namespace))
		if err == nil {
			log.Infof("%s are available in cluster", resourceType+"/"+resourceName)
			command := exec.Command("sh", "-c", fmt.Sprintf("kubectl get %s %s -n %s", resourceType, resourceName, namespace))
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit())
			noResourcesFoundMessage := fmt.Sprintf("No resources found in %s namespace", namespace)
			if strings.Contains(string(session.Wait().Out.Contents()), noResourcesFoundMessage) {
				log.Infof("Got message => {" + noResourcesFoundMessage + "} Continue looking for resource(s)")
				continue
			}
			return nil
		}
		time.Sleep(time.Duration(pollInterval) * time.Second)
	}
	return fmt.Errorf("Error: Failed to find the resource %s of type %s, timeout reached", resourceName, resourceType)
}

// Run a command, passing through stdout/stderr to the OS standard streams
func runCommandPassThrough(env []string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	if len(env) > 0 {
		cmd.Env = env
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandPassThroughWithoutOutput(env []string, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	if len(env) > 0 {
		cmd.Env = env
	}
	return cmd.Run()
}

func VerifyControllersInCluster(namespace string) {
	Expect(waitForResource("deploy", "helm-controller", namespace, INSTALL_PODS_READY_TIMEOUT))
	Expect(waitForResource("deploy", "kustomize-controller", namespace, INSTALL_PODS_READY_TIMEOUT))
	Expect(waitForResource("deploy", "notification-controller", namespace, INSTALL_PODS_READY_TIMEOUT))
	Expect(waitForResource("deploy", "source-controller", namespace, INSTALL_PODS_READY_TIMEOUT))
	Expect(waitForResource("deploy", "image-automation-controller", namespace, INSTALL_PODS_READY_TIMEOUT))
	Expect(waitForResource("deploy", "image-reflector-controller", namespace, INSTALL_PODS_READY_TIMEOUT))
	Expect(waitForResource("pods", "", namespace, INSTALL_PODS_READY_TIMEOUT))

	By("And I wait for the wego controllers to be ready", func() {
		command := exec.Command("sh", "-c", fmt.Sprintf("kubectl wait --for=condition=Ready --timeout=%s -n %s --all pod", "120s", namespace))
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session, INSTALL_PODS_READY_TIMEOUT).Should(gexec.Exit())
	})
}

func deleteRepo(appRepoName string) {
	log.Infof("Delete application repo: %s", GITHUB_ORG+"/"+appRepoName)
	_ = runCommandPassThrough([]string{}, "hub", "delete", "-y", GITHUB_ORG+"/"+appRepoName)
}

func deleteWorkload(workloadName string, workloadNamespace string) {
	log.Infof("Delete the namespace %s along with workload %s", workloadNamespace, workloadName)
	_ = runCommandPassThrough([]string{}, "kubectl", "delete", "ns", workloadNamespace)
	_ = waitForNamespaceToTerminate(workloadNamespace, INSTALL_RESET_TIMEOUT)
}

func runCommandAndReturnOutput(commandToRun string) (stdOut string, stdErr string) {
	command := exec.Command("sh", "-c", commandToRun)
	session, _ := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Eventually(session).Should(gexec.Exit())
	return string(session.Wait().Out.Contents()), string(session.Wait().Err.Contents())
}

func generateTestInputs() TestInputs {
	var inputs TestInputs
	uniqueSuffix := RandString(6)
	inputs.appRepoName = "wego-test-app-" + RandString(8)
	inputs.appManifestFilePath = getUniqueWorkload("xxyyzz", uniqueSuffix)
	inputs.workloadName = "nginx-" + uniqueSuffix
	inputs.workloadNamespace = "my-nginx-" + uniqueSuffix
	return inputs
}

func initAndCreateEmptyRepo(appRepoName string, IsPrivateRepo bool) string {
	repoAbsolutePath := "/tmp/" + appRepoName
	privateRepo := ""
	if IsPrivateRepo {
		privateRepo = "-p"
	}
	command := exec.Command("sh", "-c", fmt.Sprintf(`
                            mkdir %s &&
                            cd %s &&
                            git init &&
                            hub create %s %s &&
							sleep 10s`, repoAbsolutePath, repoAbsolutePath, GITHUB_ORG+"/"+appRepoName, privateRepo))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit())
	return repoAbsolutePath
}

func createSubDir(subDirName string, repoAbsolutePath string) string {
	subDirAbsolutePath := fmt.Sprintf(`%s/%s`, repoAbsolutePath, subDirName)
	command := exec.Command("sh", "-c", fmt.Sprintf(`mkdir -p %s`, subDirAbsolutePath))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit())
	return subDirAbsolutePath
}

func gitAddCommitPush(repoAbsolutePath string, appManifestFilePath string) {
	command := exec.Command("sh", "-c", fmt.Sprintf(`
                            cp -r %s %s &&
                            cd %s &&
                            git add . &&
                            git commit -m 'add workload manifest' &&
                            git push -u origin main`, appManifestFilePath, repoAbsolutePath, repoAbsolutePath))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit())
}

func pullBranch(repoAbsolutePath string, branch string) {
	command := exec.Command("sh", "-c", fmt.Sprintf(`
                            cd %s &&
                            git pull origin %s`, repoAbsolutePath, branch))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit())
}

func getRepoVisibility(org string, repo string) string {
	command := exec.Command("sh", "-c", fmt.Sprintf("hub api --flat repos/%s/%s|grep -i private|cut -f2", org, repo))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit())
	visibilityStr := strings.TrimSpace(string(session.Wait().Out.Contents()))
	log.Infof("Repo visibility private=%s", visibilityStr)
	return visibilityStr
}

func pullGitRepo(repoAbsolutePath string) {
	command := exec.Command("sh", "-c", fmt.Sprintf("cd %s && git pull", repoAbsolutePath))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit())
}

func setupSSHKey(sshKeyPath string) {
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		command := exec.Command("sh", "-c", fmt.Sprintf(`
                            echo "%s" >> %s &&
                            chmod 0600 %s &&
                            ls -la %s`, os.Getenv("GITHUB_KEY"), sshKeyPath, sshKeyPath, sshKeyPath))
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit())
	}
}

func installAndVerifyWego(wegoNamespace string) {
	By("And I run 'wego install' command with namespace "+wegoNamespace, func() {
		command := exec.Command("sh", "-c", fmt.Sprintf("%s gitops install --namespace=%s", WEGO_BIN_PATH, wegoNamespace))
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session, TIMEOUT_TWO_MINUTES).Should(gexec.Exit())
		VerifyControllersInCluster(wegoNamespace)
	})
}

func runWegoAddCommand(repoAbsolutePath string, addCommand string, wegoNamespace string) {
	log.Infof("Add command to run: %s in namespace %s from dir %s", addCommand, wegoNamespace, repoAbsolutePath)
	_, _ = runWegoAddCommandWithOutput(repoAbsolutePath, addCommand, wegoNamespace)
}

func runWegoAddCommandWithOutput(repoAbsolutePath string, addCommand string, wegoNamespace string) (string, string) {
	command := exec.Command("sh", "-c", fmt.Sprintf("cd %s && %s %s", repoAbsolutePath, WEGO_BIN_PATH, addCommand))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit())
	return string(session.Wait().Out.Contents()), string(session.Wait().Err.Contents())
}

func verifyWegoAddCommand(appName string, wegoNamespace string) {
	command := exec.Command("sh", "-c", fmt.Sprintf(" kubectl wait --for=condition=Ready --timeout=60s -n %s GitRepositories --all", wegoNamespace))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session, INSTALL_PODS_READY_TIMEOUT).Should(gexec.Exit())
	Expect(waitForResource("GitRepositories", appName, wegoNamespace, INSTALL_PODS_READY_TIMEOUT)).To(Succeed())
}

func verifyWegoAddCommandWithDryRun(appRepoName string, wegoNamespace string) {
	command := exec.Command("sh", "-c", fmt.Sprintf(" kubectl wait --for=condition=Ready --timeout=30s -n %s GitRepositories --all", wegoNamespace))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session, INSTALL_PODS_READY_TIMEOUT).Should(gexec.Exit())
	Expect(waitForResource("GitRepositories", appRepoName, wegoNamespace, 30*time.Second)).ToNot(Succeed())
}

func verifyWorkloadIsDeployed(workloadName string, workloadNamespace string) {
	Expect(waitForResource("deploy", workloadName, workloadNamespace, INSTALL_PODS_READY_TIMEOUT)).To(Succeed())
	Expect(waitForResource("pods", "", workloadNamespace, INSTALL_PODS_READY_TIMEOUT)).To(Succeed())
	command := exec.Command("sh", "-c", fmt.Sprintf("kubectl wait --for=condition=Ready --timeout=60s -n %s --all pods", workloadNamespace))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session, INSTALL_PODS_READY_TIMEOUT).Should(gexec.Exit())
}

func createGitRepoBranch(repoAbsolutePath string, branchName string) string {
	command := exec.Command("sh", "-c", fmt.Sprintf("cd %s && git checkout -b %s && git push --set-upstream origin %s", repoAbsolutePath, branchName, branchName))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit())
	return string(session.Wait().Out.Contents())
}

func getUniqueWorkload(placeHolderSuffix string, uniqueSuffix string) string {
	workloadTemplateFilePath := "./data/nginx-template.yaml"
	absWorkloadManifestFilePath := "/tmp/nginx-" + uniqueSuffix + ".yaml"
	_ = runCommandPassThrough([]string{}, "sh", "-c", fmt.Sprintf("sed 's/%s/%s/g' %s > %s", placeHolderSuffix, uniqueSuffix, workloadTemplateFilePath, absWorkloadManifestFilePath))
	return absWorkloadManifestFilePath
}
