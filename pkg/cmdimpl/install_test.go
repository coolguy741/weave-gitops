package cmdimpl

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/weave-gitops/pkg/fluxops"
	"github.com/weaveworks/weave-gitops/pkg/fluxops/fluxopsfakes"
	"github.com/weaveworks/weave-gitops/pkg/shims"
)

var _ = Describe("Run Command Test", func() {
	It("Verify path through flux commands", func() {
		By("Mocking the result", func() {
			fakeHandler := &fluxopsfakes.FakeFluxHandler{
				HandleStub: func(args string) ([]byte, error) {
					return []byte("manifests"), nil
				},
			}
			fluxops.SetFluxHandler(fakeHandler)

			_, err := Install(InstallParamSet{Namespace: "my-namespace"})
			Expect(err).To(BeNil())

			args := fakeHandler.HandleArgsForCall(0)
			Expect(args).To(Equal("install --namespace=my-namespace --export"))
		})
	})
})

var _ = Describe("Exit Path Test", func() {
	It("Verify that exit is called with expected code", func() {
		By("Executing a code path that contains checkError", func() {
			exitCode := -1
			shims.WithExitHandler(localExitHandler{action: func(code int) { exitCode = code }},
				func() {
					checkError("An error message", fmt.Errorf("An error"))
				})
			Expect(exitCode).To(Equal(1))
		})
	})
})
