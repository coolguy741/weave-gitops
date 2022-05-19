package server_test

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/weave-gitops/core/clustersmngr/clustersmngrfakes"
	"github.com/weaveworks/weave-gitops/core/server"
	pb "github.com/weaveworks/weave-gitops/pkg/api/core"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	"github.com/weaveworks/weave-gitops/pkg/kube/kubefakes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetFeatureFlags(t *testing.T) {
	RegisterFailHandler(Fail)

	tests := []struct {
		name     string
		envSet   func()
		envUnset func()
		state    []client.Object
		result   map[string]string
	}{
		{
			name:     "Cluster auth secret set",
			envSet:   func() {},
			envUnset: func() {},
			state:    []client.Object{&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "flux-system", Name: "cluster-user-auth"}}},
			result: map[string]string{
				"CLUSTER_USER_AUTH": "true",
				"OIDC_AUTH":         "false",
			},
		},
		{
			name:     "Cluster auth secret not set",
			envSet:   func() {},
			envUnset: func() {},
			state:    []client.Object{},
			result: map[string]string{
				"CLUSTER_USER_AUTH": "false",
				"OIDC_AUTH":         "false",
			},
		},
		{
			name:     "OIDC secret set",
			envSet:   func() {},
			envUnset: func() {},
			state:    []client.Object{&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: "flux-system", Name: "oidc-auth"}}},
			result: map[string]string{
				"CLUSTER_USER_AUTH": "false",
				"OIDC_AUTH":         "true",
			},
		},
		{
			name:     "OIDC secret not set",
			envSet:   func() {},
			envUnset: func() {},
			state:    []client.Object{},
			result: map[string]string{
				"CLUSTER_USER_AUTH": "false",
				"OIDC_AUTH":         "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := server.NewCoreConfig(logr.Discard(), &rest.Config{}, "test", &clustersmngrfakes.FakeClientsFactory{})

			k8s := fake.NewClientBuilder().WithScheme(kube.CreateScheme()).WithObjects(tt.state...).Build()
			fakeClientGetter := kubefakes.NewFakeClientGetter(k8s)
			coreSrv, err := server.NewCoreServer(cfg, server.WithClientGetter(fakeClientGetter))
			Expect(err).NotTo(HaveOccurred())

			tt.envSet()
			defer tt.envUnset()

			resp, err := coreSrv.GetFeatureFlags(context.Background(), &pb.GetFeatureFlagsRequest{})
			Expect(err).NotTo(HaveOccurred())
			Expect(tt.result).To(Equal(resp.Flags))
		})
	}
}
