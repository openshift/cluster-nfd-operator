/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package networkpolicy

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"github.com/openshift/cluster-nfd-operator/internal/client"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("SetMasterNetworkPolicyAsDesired", func() {
	var npAPI NetworkPolicyAPI

	BeforeEach(func() {
		npAPI = NewNetworkPolicyAPI(nil, scheme)
	})

	It("should populate master network policy with correct values", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		np := networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nfd-master",
				Namespace: "test-namespace",
			},
		}

		err := npAPI.SetMasterNetworkPolicyAsDesired(&nfdCR, &np)
		Expect(err).To(BeNil())

		Expect(np.Spec.PodSelector.MatchLabels).To(Equal(map[string]string{"app": "nfd-master"}))
		Expect(np.Spec.PolicyTypes).To(Equal([]networkingv1.PolicyType{networkingv1.PolicyTypeIngress}))
		Expect(np.Spec.Ingress).To(HaveLen(1))

		httpPort := intstr.FromInt32(8080)
		tcp := corev1.ProtocolTCP
		Expect(np.Spec.Ingress[0].Ports).To(Equal([]networkingv1.NetworkPolicyPort{
			{Protocol: &tcp, Port: &httpPort},
		}))
	})
})

var _ = Describe("SetWorkerNetworkPolicyAsDesired", func() {
	var npAPI NetworkPolicyAPI

	BeforeEach(func() {
		npAPI = NewNetworkPolicyAPI(nil, scheme)
	})

	It("should populate worker network policy with deny-all ingress", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		np := networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nfd-worker",
				Namespace: "test-namespace",
			},
		}

		err := npAPI.SetWorkerNetworkPolicyAsDesired(&nfdCR, &np)
		Expect(err).To(BeNil())

		Expect(np.Spec.PodSelector.MatchLabels).To(Equal(map[string]string{"app": "nfd-worker"}))
		Expect(np.Spec.PolicyTypes).To(Equal([]networkingv1.PolicyType{networkingv1.PolicyTypeIngress}))
		Expect(np.Spec.Ingress).To(BeEmpty())
	})
})

var _ = Describe("SetGCNetworkPolicyAsDesired", func() {
	var npAPI NetworkPolicyAPI

	BeforeEach(func() {
		npAPI = NewNetworkPolicyAPI(nil, scheme)
	})

	It("should populate gc network policy with correct values", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		np := networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nfd-gc",
				Namespace: "test-namespace",
			},
		}

		err := npAPI.SetGCNetworkPolicyAsDesired(&nfdCR, &np)
		Expect(err).To(BeNil())

		Expect(np.Spec.PodSelector.MatchLabels).To(Equal(map[string]string{"app": "nfd-gc"}))
		Expect(np.Spec.PolicyTypes).To(Equal([]networkingv1.PolicyType{networkingv1.PolicyTypeIngress}))
		Expect(np.Spec.Ingress).To(HaveLen(1))

		httpPort := intstr.FromInt32(8080)
		tcp := corev1.ProtocolTCP
		Expect(np.Spec.Ingress[0].Ports).To(Equal([]networkingv1.NetworkPolicyPort{
			{Protocol: &tcp, Port: &httpPort},
		}))
	})
})

var _ = Describe("DeleteNetworkPolicy", func() {
	var (
		ctrl  *gomock.Controller
		clnt  *client.MockClient
		npAPI NetworkPolicyAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)
		npAPI = NewNetworkPolicyAPI(clnt, scheme)
	})

	ctx := context.Background()
	name := "np-name"
	namespace := "np-namespace"
	expectedNP := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	It("failure to delete network policy from the cluster", func() {
		clnt.EXPECT().Delete(ctx, expectedNP).Return(fmt.Errorf("some error"))

		err := npAPI.DeleteNetworkPolicy(ctx, namespace, name)
		Expect(err).To(HaveOccurred())
	})

	It("network policy is not present in the cluster", func() {
		clnt.EXPECT().Delete(ctx, expectedNP).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever"))

		err := npAPI.DeleteNetworkPolicy(ctx, namespace, name)
		Expect(err).To(BeNil())
	})

	It("network policy deleted successfully", func() {
		clnt.EXPECT().Delete(ctx, expectedNP).Return(nil)

		err := npAPI.DeleteNetworkPolicy(ctx, namespace, name)
		Expect(err).To(BeNil())
	})
})
