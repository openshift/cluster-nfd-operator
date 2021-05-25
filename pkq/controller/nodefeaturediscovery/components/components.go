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
)

const (
	// AssetsDir defines the directory with assets under the operator image
	AssetsDir = "/assets"
)

// Get the NFD operator's service account
func GetServiceAccount(nfd *nfdv1.NodeFeatureDiscovery) (map[string]string, error) {
	if nfd.Spec.ServiceAccount != nil {
		return nfd.Spec.ServiceAccount, nil
	}
	return nil, errors.New("Could not find ServiceAccount")
}

// Get the NFD operator's cluster role
func GetClusterRole(nfd *nfdv1.NodeFeatureDiscovery) (map[string]string, error) {
	if nfd.Spec.ClusterRole != nil {
		return nfd.Spec.ClusterRole, nil
	}
	return nil, errors.New("Could not find ClusterRole")
}

// Get the NFD operator's cluster role binding
func GetClusterRoleBinding(nfd *nfdv1.NodeFeatureDiscovery) (map[string]string, error) {
	if nfd.Spec.ClusterRoleBinding != nil {
		return nfd.Spec.ClusterRoleBinding, nil
	}
	return nil, errors.New("Could not find Cluster RoleBinding")
}

// Get the NFD operator's daemon set
func GetDaemonSet(nfd *nfdv1.NodeFeatureDiscovery) (map[string]string, error) {
	if nfd.Spec.DaemonSet != nil {
		return nfd.Spec.DaemonSet, nil
	}
	return nil, errors.New("Could not find DaemonSet")
}

// Get the NFD operator's service
func GetService(nfd *nfdv1.NodeFeatureDiscovery) (map[string]string, error) {
	if nfd.Spec.Service != nil {
		return nfd.Spec.Service, nil
	}
	return nil, errors.New("Could not find Service")
}
