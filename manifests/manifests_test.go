package manifests

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing GenerateManifests", func() {
	It("should contain all the manifests for Wego", func() {
		params := Params{
			AppVersion:      "latest",
			ProfilesVersion: "0.1.0",
			Namespace:       "my-namespace",
		}

		manifestsBytes, err := GenerateManifests(params)
		Expect(err).NotTo(HaveOccurred())

		var manifests string
		for _, m := range manifestsBytes {
			manifests = manifests + string(m)
		}

		By("containing the App API manifests", func() {
			By("containing a Deployment manifest")
			Expect(manifests).To(ContainSubstring(`
kind: Deployment
metadata:
  name: wego-app
  namespace: my-namespace`))
			Expect(manifests).To(ContainSubstring("latest"))

			By("containing a Service manifest")
			Expect(manifests).To(ContainSubstring(`
kind: Service
metadata:
  name: wego-app
  namespace: my-namespace`))

			By("containing a Service Account manifest")
			Expect(manifests).To(ContainSubstring(`
kind: ServiceAccount
metadata:
  name: wego-app-service-account
  namespace: my-namespace`))

			By("containing a Role manifest")
			Expect(manifests).To(ContainSubstring(`
kind: Role
metadata:
  name: resources-reader`))

			By("containing a Role Binding manifest")
			Expect(manifests).To(ContainSubstring(`
kind: RoleBinding
metadata:
  name: read-resources`))
		})
	})
})
