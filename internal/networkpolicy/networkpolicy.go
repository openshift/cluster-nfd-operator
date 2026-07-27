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

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
)

//go:generate mockgen -source=networkpolicy.go -package=networkpolicy -destination=mock_networkpolicy.go NetworkPolicyAPI

type NetworkPolicyAPI interface {
	SetMasterNetworkPolicyAsDesired(nfdInstance *nfdv1.NodeFeatureDiscovery, np *networkingv1.NetworkPolicy) error
	SetWorkerNetworkPolicyAsDesired(nfdInstance *nfdv1.NodeFeatureDiscovery, np *networkingv1.NetworkPolicy) error
	SetGCNetworkPolicyAsDesired(nfdInstance *nfdv1.NodeFeatureDiscovery, np *networkingv1.NetworkPolicy) error
	DeleteNetworkPolicy(ctx context.Context, namespace, name string) error
}

type networkPolicy struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewNetworkPolicyAPI(client client.Client, scheme *runtime.Scheme) NetworkPolicyAPI {
	return &networkPolicy{
		client: client,
		scheme: scheme,
	}
}

func (n *networkPolicy) SetMasterNetworkPolicyAsDesired(nfdInstance *nfdv1.NodeFeatureDiscovery, np *networkingv1.NetworkPolicy) error {
	podSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "nfd-master"},
	}
	np.Spec = networkingv1.NetworkPolicySpec{
		PodSelector: podSelector,
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		Ingress:     healthProbeIngressRules(),
		Egress:      requiredEgressRules(),
	}
	return controllerutil.SetControllerReference(nfdInstance, np, n.scheme)
}

func (n *networkPolicy) SetWorkerNetworkPolicyAsDesired(nfdInstance *nfdv1.NodeFeatureDiscovery, np *networkingv1.NetworkPolicy) error {
	podSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "nfd-worker"},
	}
	np.Spec = networkingv1.NetworkPolicySpec{
		PodSelector: podSelector,
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		Ingress:     []networkingv1.NetworkPolicyIngressRule{},
		Egress:      requiredEgressRules(),
	}
	return controllerutil.SetControllerReference(nfdInstance, np, n.scheme)
}

func (n *networkPolicy) SetGCNetworkPolicyAsDesired(nfdInstance *nfdv1.NodeFeatureDiscovery, np *networkingv1.NetworkPolicy) error {
	podSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "nfd-gc"},
	}
	np.Spec = networkingv1.NetworkPolicySpec{
		PodSelector: podSelector,
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
		Ingress:     healthProbeIngressRules(),
		Egress:      requiredEgressRules(),
	}
	return controllerutil.SetControllerReference(nfdInstance, np, n.scheme)
}

func (n *networkPolicy) DeleteNetworkPolicy(ctx context.Context, namespace, name string) error {
	np := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
	err := n.client.Delete(ctx, &np)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete NetworkPolicy %s/%s: %w", namespace, name, err)
	}
	return nil
}

func healthProbeIngressRules() []networkingv1.NetworkPolicyIngressRule {
	httpPort := intstr.FromInt32(8080)
	tcp := corev1.ProtocolTCP
	return []networkingv1.NetworkPolicyIngressRule{
		{
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: &tcp,
					Port:     &httpPort,
				},
			},
		},
	}
}

// requiredEgressRules returns the minimal egress ports needed by every NFD
// component: DNS resolution and kube-apiserver communication. Rules are
// port-only (no destination/peer restriction) because both the apiserver and
// CoreDNS on OpenShift are host-networked pods, whose matching behaviour in
// namespace/pod selectors is implementation-defined and unreliable for egress
// targeting on OVN-Kubernetes. Port-only rules are the pattern used by
// upstream node-feature-discovery's Helm chart.
func requiredEgressRules() []networkingv1.NetworkPolicyEgressRule {
	tcp := corev1.ProtocolTCP
	udp := corev1.ProtocolUDP
	httpsPort := intstr.FromInt32(443)
	apiServerPort := intstr.FromInt32(6443)
	dnsPort := intstr.FromInt32(53)
	// OVN-Kubernetes evaluates egress rules after Service DNAT, so the
	// destination port is the pod's targetPort (5353), not the Service
	// port (53). Both are needed: 53 for compatibility with other CNIs,
	// 5353 for OVN-Kubernetes on OpenShift.
	dnsTargetPort := intstr.FromInt32(5353)

	return []networkingv1.NetworkPolicyEgressRule{
		{
			Ports: []networkingv1.NetworkPolicyPort{
				{Protocol: &tcp, Port: &httpsPort},
				{Protocol: &tcp, Port: &apiServerPort},
				{Protocol: &tcp, Port: &dnsPort},
				{Protocol: &udp, Port: &dnsPort},
				{Protocol: &tcp, Port: &dnsTargetPort},
				{Protocol: &udp, Port: &dnsTargetPort},
			},
		},
	}
}
