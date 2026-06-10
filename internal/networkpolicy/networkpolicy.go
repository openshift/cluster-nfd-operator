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
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		Ingress:     healthProbeIngressRules(),
	}
	return controllerutil.SetControllerReference(nfdInstance, np, n.scheme)
}

func (n *networkPolicy) SetWorkerNetworkPolicyAsDesired(nfdInstance *nfdv1.NodeFeatureDiscovery, np *networkingv1.NetworkPolicy) error {
	podSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "nfd-worker"},
	}
	np.Spec = networkingv1.NetworkPolicySpec{
		PodSelector: podSelector,
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		Ingress:     []networkingv1.NetworkPolicyIngressRule{},
	}
	return controllerutil.SetControllerReference(nfdInstance, np, n.scheme)
}

func (n *networkPolicy) SetGCNetworkPolicyAsDesired(nfdInstance *nfdv1.NodeFeatureDiscovery, np *networkingv1.NetworkPolicy) error {
	podSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "nfd-gc"},
	}
	np.Spec = networkingv1.NetworkPolicySpec{
		PodSelector: podSelector,
		PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		Ingress:     healthProbeIngressRules(),
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
