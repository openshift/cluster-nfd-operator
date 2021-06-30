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

	// Error messages
	errorCouldNotFindWorkerDaemonSet = "Could not find NFD Operator Worker DaemonSet"
	errorCouldNotFindMasterDaemonSet = "Could not find NFD Operator Master DaemonSet"
	errorCouldNotFindService         = "Could not find NFD Operator Service"
	errorCouldNotFindWorkerConfig    = "Could not find NFD Operator Worker Config"
	errorCouldNotFindRole            = "Could not find NFD Operator Role"
	errorCouldNotFindRoleBinding     = "Could not find NFD Operator Role Binding"
	errorCouldNotFindServiceAccount  = "Could not find NFD Operator Service Account"
)

//// Get the NFD operator's pods
//func GetPod(nfd *nfdv1.NodeFeatureDiscovery) (*corev1.Pod, error) {
//	if &nfd.Spec.Pod != nil {
//		return &nfd.Spec.Pod, nil
//	}
//	return nil, errors.New("Could not find Pod")
//}

// Get the NFD operator's worker daemon set
func GetWorkerDaemonSet(instance *nfdv1.NodeFeatureDiscovery) (*appsv1.DaemonSet, error) {

	// Initialize the error to 'nil' so that it's easy to keep track
	// of the error messages
	var err error = nil

	// Index the Worker DaemonSet since it will be referenced often
	// in this function
	var ds = instance.Spec.WorkerDaemonSet

	// If the Worker DaemonSet object is empty, try to find it in
	// the 'NFD' object
	if ds == nil {
		err = errors.New(errorCouldNotFindWorkerDaemonSet)
	}
	return ds, err
}

// Get the NFD operator's master daemon set
func GetMasterDaemonSet(nfd *nfdv1.NodeFeatureDiscovery) (*appsv1.DaemonSet, error) {
	var err error = nil
	if nfd.Spec.MasterDaemonSet == nil {
		err = errors.New(errorCouldNotFindMasterDaemonSet)
	}
	return nfd.Spec.MasterDaemonSet, err
}

// Get the NFD operator's service
func GetService(nfd *nfdv1.NodeFeatureDiscovery) (*corev1.Service, error) {
	var err error = nil
	if nfd.Spec.Service == nil {
		err = errors.New(errorCouldNotFindService)
	}
	return nfd.Spec.Service, err
}

// Get the NFD operator's worker config
func GetWorkerConfig(nfd *nfdv1.NodeFeatureDiscovery) (*nfdv1.ConfigMap, error) {
	var err error = nil
	if nfd.Spec.WorkerConfig == nil {
		err = errors.New(errorCouldNotFindWorkerConfig)
	}
	return nfd.Spec.WorkerConfig, err
}

// Get the NFD operator's role
func GetRole(nfd *nfdv1.NodeFeatureDiscovery) (*rbacv1.Role, error) {
	var err error = nil
	if nfd.Spec.Role == nil {
		err = errors.New(errorCouldNotFindRole)
	}
	return nfd.Spec.Role, err
}

// Get the NFD operator's role binding
func GetRoleBinding(nfd *nfdv1.NodeFeatureDiscovery) (*rbacv1.RoleBinding, error) {
	var err error = nil
	if nfd.Spec.RoleBinding == nil {
		err = errors.New(errorCouldNotFindRoleBinding)
	}
	return nfd.Spec.RoleBinding, err
}

// Get the NFD operator's service account
func GetServiceAccount(nfd *nfdv1.NodeFeatureDiscovery) (*corev1.ServiceAccount, error) {
	var err error = nil
	if nfd.Spec.ServiceAccount == nil {
		err = errors.New(errorCouldNotFindServiceAccount)
	}
	return nfd.Spec.ServiceAccount, err
}
