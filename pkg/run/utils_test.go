package run

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/weaveworks/weave-gitops/pkg/kube"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type isLocalClusterTest struct {
	clusterName string
	expected    bool
}

func runIsLocalClusterTest(client *kube.KubeHTTP, test isLocalClusterTest) {
	actual := IsLocalCluster(client)

	Expect(actual).To(Equal(test.expected))
}

var _ = Describe("IsLocalCluster", func() {
	var fakeKube *kube.KubeHTTP

	BeforeEach(func() {
		fakeKube = &kube.KubeHTTP{}
	})

	It("returns true for kind prefix", func() {
		test := isLocalClusterTest{
			clusterName: "kind-wego-dev",
			expected:    true,
		}

		fakeKube.ClusterName = test.clusterName

		runIsLocalClusterTest(fakeKube, test)
	})

	It("returns true for k3d prefix", func() {
		test := isLocalClusterTest{
			clusterName: "k3d-wego-dev",
			expected:    true,
		}

		fakeKube.ClusterName = test.clusterName

		runIsLocalClusterTest(fakeKube, test)
	})

	It("returns true if cluster name is minikube", func() {
		test := isLocalClusterTest{
			clusterName: "minikube",
			expected:    true,
		}

		fakeKube.ClusterName = test.clusterName

		runIsLocalClusterTest(fakeKube, test)
	})

	It("returns true if cluster name is docker-for-desktop", func() {
		test := isLocalClusterTest{
			clusterName: "docker-for-desktop",
			expected:    true,
		}

		fakeKube.ClusterName = test.clusterName

		runIsLocalClusterTest(fakeKube, test)
	})

	It("returns true if cluster name is docker-desktop", func() {
		test := isLocalClusterTest{
			clusterName: "docker-desktop",
			expected:    true,
		}

		fakeKube.ClusterName = test.clusterName

		runIsLocalClusterTest(fakeKube, test)
	})

	It("returns false for a gke cluster", func() {
		test := isLocalClusterTest{
			clusterName: "gke_testing_cluster-1",
			expected:    false,
		}

		fakeKube.ClusterName = test.clusterName

		runIsLocalClusterTest(fakeKube, test)
	})

	It("returns false for an empty string", func() {
		test := isLocalClusterTest{
			clusterName: "",
			expected:    false,
		}

		fakeKube.ClusterName = test.clusterName

		runIsLocalClusterTest(fakeKube, test)
	})
})

// mock controller-runtime client
type mockClientForGetPodFromResourceDescription struct {
	client.Client
	state stateGetPodFromResourceDescription
}

type stateGetPodFromResourceDescription string

const (
	stateListReturnErr    stateGetPodFromResourceDescription = "list-return-err"
	stateListNoRunningPod stateGetPodFromResourceDescription = "list-no-running-pod"
	stateListZeroPod      stateGetPodFromResourceDescription = "list-zero-pod"
	stateListHasPod       stateGetPodFromResourceDescription = "list-has-pod"

	stateGetReturnErr stateGetPodFromResourceDescription = "get-return-err"
)

func (c *mockClientForGetPodFromResourceDescription) List(_ context.Context, list client.ObjectList, opts ...client.ListOption) error {
	switch c.state {
	case stateListReturnErr:
		return errors.New("fake error")

	default:
		listOptions := &client.ListOptions{}
		for _, opt := range opts {
			opt.ApplyToList(listOptions)
		}

		podList := &corev1.PodList{}

		switch c.state {
		case stateListZeroPod:
			podList.Items = []corev1.Pod{}

		case stateListNoRunningPod:
			podList.Items = append(podList.Items, corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: listOptions.Namespace,
				},
			})

		case stateListHasPod:
			podList.Items = append(podList.Items, corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-1",
					Namespace: listOptions.Namespace,
				},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{},
					Phase:      corev1.PodRunning,
				},
			})
		}

		podList.DeepCopyInto(list.(*corev1.PodList))
	}

	return nil
}

func (c *mockClientForGetPodFromResourceDescription) Get(_ context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	switch c.state {
	case stateGetReturnErr:
		return errors.New("fake error")

	default:
		switch obj := obj.(type) {
		case *corev1.Pod:
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
			}
			pod.DeepCopyInto(obj)
		case *corev1.Service:
			service := corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"app": key.Name,
					},
				},
			}
			service.DeepCopyInto(obj)
		case *appsv1.Deployment:
			deployment := appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": key.Name,
						},
					},
				},
			}
			deployment.DeepCopyInto(obj)
		}
	}

	return nil
}

var _ = Describe("GetPodFromResourceDescription", func() {
	It("should return an error if the pod spec is not correct", func() {
		namespacedName := types.NamespacedName{Namespace: "", Name: ""}

		_, err := GetPodFromResourceDescription(context.Background(), namespacedName, "something", &mockClientForGetPodFromResourceDescription{})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unsupported spec kind"))
	})

	It("should return an error if the client returns an error", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		_, err := GetPodFromResourceDescription(context.Background(), namespacedName, "pod", &mockClientForGetPodFromResourceDescription{state: stateGetReturnErr})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake error"))
	})

	It("returns a pod according to the pod spec", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		pod, err := GetPodFromResourceDescription(context.Background(), namespacedName, "pod", &mockClientForGetPodFromResourceDescription{})

		Expect(err).To(BeNil())
		Expect(pod.Name).To(Equal("name"))
		Expect(pod.Namespace).To(Equal("ns"))
	})

	// Service tests

	It("should return an error if the client returns an error", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		_, err := GetPodFromResourceDescription(context.Background(), namespacedName, "service", &mockClientForGetPodFromResourceDescription{state: stateGetReturnErr})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("error getting service: fake error, namespaced Name: ns/name"))
	})

	It("should return an error if the client returns an error", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		_, err := GetPodFromResourceDescription(context.Background(), namespacedName, "service", &mockClientForGetPodFromResourceDescription{state: stateListReturnErr})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake error"))
	})

	It("returns a pod according to the service spec", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		pod, err := GetPodFromResourceDescription(context.Background(), namespacedName, "service", &mockClientForGetPodFromResourceDescription{state: stateListHasPod})

		Expect(err).To(BeNil())
		Expect(pod.Name).To(Equal("pod-1"))
		Expect(pod.Namespace).To(Equal("ns"))
	})

	It("returns a pod according to the service spec", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		pod, err := GetPodFromResourceDescription(context.Background(), namespacedName, "service", &mockClientForGetPodFromResourceDescription{state: stateListZeroPod})

		Expect(err).To(HaveOccurred())
		Expect(pod).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("no pods found for service"))
	})

	It("returns a pod according to the service spec", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		pod, err := GetPodFromResourceDescription(context.Background(), namespacedName, "service", &mockClientForGetPodFromResourceDescription{state: stateListNoRunningPod})

		Expect(err).To(HaveOccurred())
		Expect(pod).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("no running pods found for service"))
	})

	// Deployment tests

	It("should return an error if the client returns an error", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		_, err := GetPodFromResourceDescription(context.Background(), namespacedName, "deployment", &mockClientForGetPodFromResourceDescription{state: stateGetReturnErr})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("error getting deployment: fake error, namespaced Name: ns/name"))
	})

	It("should return an error if the client returns an error", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		_, err := GetPodFromResourceDescription(context.Background(), namespacedName, "deployment", &mockClientForGetPodFromResourceDescription{state: stateListReturnErr})

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake error"))
	})

	It("returns a pod according to the deployment spec", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		pod, err := GetPodFromResourceDescription(context.Background(), namespacedName, "deployment", &mockClientForGetPodFromResourceDescription{state: stateListHasPod})

		Expect(err).To(BeNil())
		Expect(pod.Name).To(Equal("pod-1"))
		Expect(pod.Namespace).To(Equal("ns"))
	})

	It("returns a pod according to the deployment spec", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}

		pod, err := GetPodFromResourceDescription(context.Background(), namespacedName, "deployment", &mockClientForGetPodFromResourceDescription{state: stateListZeroPod})

		Expect(err).To(HaveOccurred())
		Expect(pod).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("no pods found for deployment"))
	})

	It("returns a pod according to the deployment spec", func() {
		namespacedName := types.NamespacedName{Namespace: "ns", Name: "name"}
		pod, err := GetPodFromResourceDescription(context.Background(), namespacedName, "deployment", &mockClientForGetPodFromResourceDescription{state: stateListNoRunningPod})

		Expect(err).To(HaveOccurred())
		Expect(pod).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("no running pods found for deployment"))
	})
})
