package server_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/weave-gitops/pkg/api/applications"
)

var _ = Describe("ApplicationsServer", func() {
	It("AddApplication", func() {
		res, err := client.ListApplications(context.Background(), &applications.ListApplicationsRequest{})

		Expect(err).NotTo(HaveOccurred())

		Expect(len(res.Applications)).To(Equal(3))
	})
})
