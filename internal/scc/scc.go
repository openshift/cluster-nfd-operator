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

package scc

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	securityv1 "github.com/openshift/api/security/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
)

//go:generate mockgen -source=scc.go -package=scc -destination=mock_scc.go SccAPI

type SccAPI interface {
	SetWorkerSCCAsDesired(ctx context.Context, nfdInstance *nfdv1.NodeFeatureDiscovery, workerSCC *securityv1.SecurityContextConstraints) error
	SetTopologySCCAsDesired(ctx context.Context, nfdInstance *nfdv1.NodeFeatureDiscovery, topologySCC *securityv1.SecurityContextConstraints) error
	DeleteSCC(ctx context.Context, sccName string) error
}

type scc struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewSccAPI(client client.Client, scheme *runtime.Scheme) SccAPI {
	return &scc{
		client: client,
		scheme: scheme,
	}
}

func (s *scc) SetWorkerSCCAsDesired(ctx context.Context, nfdInstance *nfdv1.NodeFeatureDiscovery, workerSCC *securityv1.SecurityContextConstraints) error {
	workerSCC.ObjectMeta.Annotations = map[string]string{
		"kubernetes.io/description": "nfd-master allows using host networking, host ports and hostPath but still requires pods to be run with a UID and SELinux context that are allocated to the namespace.",
	}
	workerSCC.AllowHostDirVolumePlugin = true
	workerSCC.AllowHostNetwork = true
	workerSCC.AllowHostPorts = true
	workerSCC.AllowPrivilegeEscalation = ptr.To(true)
	workerSCC.FSGroup = securityv1.FSGroupStrategyOptions{
		Type: securityv1.FSGroupStrategyMustRunAs,
	}
	workerSCC.RequiredDropCapabilities = []corev1.Capability{
		"KILL", "MKNOD", "SETUID", "SETGID",
	}
	workerSCC.RunAsUser = securityv1.RunAsUserStrategyOptions{
		Type: securityv1.RunAsUserStrategyMustRunAsRange,
	}
	workerSCC.SELinuxContext = securityv1.SELinuxContextStrategyOptions{
		Type: securityv1.SELinuxStrategyMustRunAs,
	}
	workerSCC.SupplementalGroups = securityv1.SupplementalGroupsStrategyOptions{
		Type: securityv1.SupplementalGroupsStrategyRunAsAny,
	}
	workerSCC.SeccompProfiles = []string{
		"*",
	}
	workerSCC.Users = []string{
		"system:serviceaccount:openshift-nfd:nfd-worker",
	}
	workerSCC.Volumes = []securityv1.FSType{
		securityv1.FSTypeConfigMap,
		securityv1.FSTypeDownwardAPI,
		securityv1.FSTypeEmptyDir,
		securityv1.FSTypePersistentVolumeClaim,
		securityv1.FSProjected,
		securityv1.FSTypeSecret,
		securityv1.FSTypeHostPath,
	}
	return nil
}

func (s *scc) SetTopologySCCAsDesired(ctx context.Context, nfdInstance *nfdv1.NodeFeatureDiscovery, topologySCC *securityv1.SecurityContextConstraints) error {
	topologySCC.ObjectMeta.Annotations = map[string]string{
		"kubernetes.io/description": "nfd-topology-updater",
	}
	topologySCC.AllowHostDirVolumePlugin = true
	topologySCC.AllowHostNetwork = true
	topologySCC.AllowHostPorts = true
	topologySCC.AllowPrivilegeEscalation = ptr.To(true)
	topologySCC.FSGroup = securityv1.FSGroupStrategyOptions{
		Type: securityv1.FSGroupStrategyRunAsAny,
	}
	topologySCC.ReadOnlyRootFilesystem = true
	topologySCC.RequiredDropCapabilities = []corev1.Capability{
		"KILL", "MKNOD", "SETUID", "SETGID",
	}
	topologySCC.RunAsUser = securityv1.RunAsUserStrategyOptions{
		Type: securityv1.RunAsUserStrategyRunAsAny,
	}
	topologySCC.SELinuxContext = securityv1.SELinuxContextStrategyOptions{
		Type: securityv1.SELinuxStrategyRunAsAny,
	}
	topologySCC.SupplementalGroups = securityv1.SupplementalGroupsStrategyOptions{
		Type: securityv1.SupplementalGroupsStrategyRunAsAny,
	}
	topologySCC.SeccompProfiles = []string{
		"*",
	}
	topologySCC.Users = []string{
		"system:serviceaccount:openshift-nfd:nfd-topology-updater",
	}
	topologySCC.Volumes = []securityv1.FSType{
		securityv1.FSTypeConfigMap,
		securityv1.FSTypeDownwardAPI,
		securityv1.FSTypeEmptyDir,
		securityv1.FSTypePersistentVolumeClaim,
		securityv1.FSProjected,
		securityv1.FSTypeSecret,
		securityv1.FSTypeHostPath,
	}
	return nil
}

func (s *scc) DeleteSCC(ctx context.Context, sccName string) error {
	sc := securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: sccName,
		},
	}
	err := s.client.Delete(ctx, &sc)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete SCC %s: %w", sccName, err)
	}
	return nil
}
