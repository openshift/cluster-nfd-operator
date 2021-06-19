/*

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

package components

import (
	"errors"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	// AssetsDir defines the directory with assets under the operator image
	AssetsDir = "/assets"
)

//// Get the NFD operator's service account
//func GetServiceAccount(nfd *nfdv1.NodeFeatureDiscovery) (*corev1.ServiceAccount, error) {
//	if &nfd.Spec.ServiceAccount != nil {
//		return &nfd.Spec.ServiceAccount, nil
//	}
//	return nil, errors.New("Could not find ServiceAccount")
//}

//// Get the NFD operator's cluster role binding
//func GetClusterRoleBinding(nfd *nfdv1.NodeFeatureDiscovery) (*rbacv1.ClusterRoleBinding, error) {
//	if &nfd.Spec.ClusterRoleBinding != nil {
//		return &nfd.Spec.ClusterRoleBinding, nil
//	}
//	return nil, errors.New("Could not find ClusterRoleBinding")
//}

//// Get the NFD operator's pods
//func GetPod(nfd *nfdv1.NodeFeatureDiscovery) (*corev1.Pod, error) {
//	if &nfd.Spec.Pod != nil {
//		return &nfd.Spec.Pod, nil
//	}
//	return nil, errors.New("Could not find Pod")
//}

// Get the NFD operator's daemon set
func GetDaemonSet(nfd *nfdv1.NodeFeatureDiscovery) (*appsv1.DaemonSet, error) {
	var err error = nil
	if nfd.Spec.DaemonSet == nil {
		err = errors.New("Could not find NFD Operator DaemonSet")
	}
	return nfd.Spec.DaemonSet, err
}

// Get the NFD operator's service
func GetService(nfd *nfdv1.NodeFeatureDiscovery) (*corev1.Service, error) {
	var err error = nil
	if nfd.Spec.Service == nil {
		err = errors.New("Could not find NFD Operator Service")
	}
	return nfd.Spec.Service, err
}

// Get the NFD operator's worker config
func GetWorkerConfig(nfd *nfdv1.NodeFeatureDiscovery) (*nfdv1.ConfigMap, error) {
	var err error = nil
	if nfd.Spec.WorkerConfig == nil {
		err = errors.New("Could not find NFD Operator Worker Config")
	}
	return nfd.Spec.WorkerConfig, err
}

// Get the NFD operator's cluster role
func GetClusterRole(nfd *nfdv1.NodeFeatureDiscovery) (*rbacv1.ClusterRole, error) {
	var err error = nil
	if nfd.Spec.ClusterRole == nil {
		err = errors.New("Could not find NFD Operator ClusterRole")
	}
	return nfd.Spec.ClusterRole, err
}
