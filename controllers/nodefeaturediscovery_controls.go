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

package controllers

import (
	"context"
	"fmt"
	"strings"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var r NodeFeatureDiscoveryReconciler

type controlFunc []func(n NFD) (ResourceStatus, error)

// ResourceStatus defines the status of the resource (0 or 1, for Ready/NotReady)
type ResourceStatus int

// Ready/NotReady defines if a resource is ready.
const (
	Ready    ResourceStatus = 0
	NotReady ResourceStatus = 1

	defaultServicePort int = 12000
)

// String returns the status of the resource as being Ready,
// NotReady, or Unknown Resource Status
func (s ResourceStatus) String() string {
	names := [...]string{
		"Ready",
		"NotReady"}

	// Ideally, 's' should be either Ready (=0) or NotReady (=1),
	// but we may run into a case where we get an unknown status,
	// so return information stating that the resource status is
	// unknown
	if s < Ready || s > NotReady {
		return "Unkown Resources Status"
	}
	return names[s]
}

// Namespace checks if the Namespace for NFD exists and attempts to create
// it if it doesn't exist
func Namespace(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// Namespace object, so let's get the resource's Namespace object
	obj := n.resources[state].Namespace

	// found states if the Namespace was found
	found := &corev1.Namespace{}

	// Look for the Namespace to see if it exists, and if so, check if
	// it's Ready/NotReady. If the Namespace does not exist, then
	// attempt to create it
	r.Log.Info("Looking for Namespace '", obj.Name, "'")
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating ")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	r.Log.Info("Found, skipping update")

	return Ready, nil
}

// ServiceAccount checks if the ServiceAccount for NFD exists and attempts to
// create it if it doesn't exist.
func ServiceAccount(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ServiceAccount object, so let's get the resource's ServiceAccount
	// object
	obj := n.resources[state].ServiceAccount

	// It is also assumed that our service account has a defined Namespace
	obj.SetNamespace(n.ins.GetNamespace())

	// found states if the ServiceAccount was found
	found := &corev1.ServiceAccount{}
	r.Log.Info("Looking for ServiceAccount '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the ServiceAccount to see if it exists, and if so, check if
	// it's Ready/NotReady. If the ServiceAccount does not exist, then
	// attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating ")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	r.Log.Info("Found, skipping update")

	return Ready, nil
}

// ClusterRole attempts to create a ClusterRole in a given Namespace. If
// the ClusterRole already exists, then attempt to update it.
func ClusterRole(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ClusterRole object, so let's get the resource's ClusterRole
	// object
	obj := n.resources[state].ClusterRole

	// found states if the ClusterRole was found
	found := &rbacv1.ClusterRole{}
	r.Log.Info("Looking for ClusterRole '", obj.Name, "'")

	// Look for the ClusterRole to see if it exists, and if so, check if
	// it's Ready/NotReady. If the ClusterRole does not exist, then
	// attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the ClusterRole, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// ClusterRoleBinding attempts to create a ClusterRoleBinding in a given
// Namespace. If the ClusterRoleBinding already exists, then attempt to
// update it.
func ClusterRoleBinding(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ClusterRoleBinding object, so let's get the resource's
	// ClusterRoleBinding object
	obj := n.resources[state].ClusterRoleBinding

	// found states if the ClusterRoleBinding was found
	found := &rbacv1.ClusterRoleBinding{}

	// It is also assumed that our ClusterRoleBinding has a defined
	// Namespace
	obj.Subjects[0].Namespace = n.ins.GetNamespace()

	r.Log.Info("Looking for ClusterRoleBinding '", obj.Name, "'")

	// Look for the ClusterRoleBinding to see if it exists, and if so,
	// check if it's Ready/NotReady. If the ClusterRoleBinding does not
	// exist, then attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the ClusterRoleBinding, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// Role attempts to create a Role in a given Namespace. If the Role
// already exists, then attempt to update it.
func Role(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// Role object, so let's get the resource's Role object
	obj := n.resources[state].Role

	// The Namespace should already be defined, so let's set the
	// namespace to the namespace defined in the Role object
	obj.SetNamespace(n.ins.GetNamespace())

	// found states if the Role was found
	found := &rbacv1.Role{}
	r.Log.Info("Looking for Role '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the Role to see if it exists, and if so, check if it's
	// Ready/NotReady. If the Role does not exist, then attempt to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the Role, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// RoleBinding attempts to create a RoleBinding in a given Namespace. If
// the RoleBinding already exists, then attempt to update it.
func RoleBinding(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// RoleBinding object, so let's get the resource's RoleBinding
	// object
	obj := n.resources[state].RoleBinding

	// The Namespace should already be defined, so let's set the
	// namespace to the namespace defined in the RoleBinding object
	obj.SetNamespace(n.ins.GetNamespace())

	// found states if the RoleBinding was found
	found := &rbacv1.RoleBinding{}
	r.Log.Info("Looking for RoleBinding", obj.Name, "in Namespace", obj.Namespace)

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the RoleBinding to see if it exists, and if so, check if
	// it's Ready/NotReady. If the RoleBinding does not exist, then attempt
	// to create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the RoleBinding, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// ConfigMap attempts to create a ConfigMap in a given Namespace. If
// the ConfigMap already exists, then attempt to update it.
func ConfigMap(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// ConfigMap object, so let's get the resource's ConfigMap object
	obj := n.resources[state].ConfigMap

	// The Namespace should already be defined, so let's set the
	// namespace to the namespace defined in the ConfigMap object
	obj.SetNamespace(n.ins.GetNamespace())

	// Update ConfigMap
	obj.ObjectMeta.Name = "nfd-worker"
	obj.Data["nfd-worker-conf"] = n.ins.Spec.WorkerConfig.ConfigData
	obj.Data["custom-conf"] = n.ins.Spec.CustomConfig.ConfigData

	// found states if the ConfigMap was found
	found := &corev1.ConfigMap{}
	r.Log.Info("Looking for ConfigMap '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the ConfigMap to see if it exists, and if so, check if it's
	// Ready/NotReady. If the ConfigMap does not exist, then attempt to create
	// it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the ConfigMap, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// DaemonSet attempts to create a DaemonSet in a given Namespace. If
// the DaemonSet already exists, then attempt to update it.
func DaemonSet(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// DaemonSet object, so let's get the resource's DaemonSet object
	obj := n.resources[state].DaemonSet

	// update the image
	obj.Spec.Template.Spec.Containers[0].Image = n.ins.Spec.Operand.ImagePath()

	// update image pull policy
	if n.ins.Spec.Operand.ImagePullPolicy != "" {
		obj.Spec.Template.Spec.Containers[0].ImagePullPolicy = n.ins.Spec.Operand.ImagePolicy(n.ins.Spec.Operand.ImagePullPolicy)
	}

	// update nfd-master service port
	if obj.ObjectMeta.Name == "nfd-master" {
		var args []string
		port := defaultServicePort
		if n.ins.Spec.Operand.ServicePort != 0 {
			port = n.ins.Spec.Operand.ServicePort
		}
		args = append(args, fmt.Sprintf("--port=%d", port))

		// check if running as instance
		// https://kubernetes-sigs.github.io/node-feature-discovery/v0.8/advanced/master-commandline-reference.html#-instance
		if n.ins.Spec.Instance != "" {
			args = append(args, fmt.Sprintf("--instance=%s", n.ins.Spec.Instance))
		}

		if len(n.ins.Spec.ExtraLabelNs) != 0 {
			args = append(args, fmt.Sprintf("--extra-label-ns=%s", strings.Join(n.ins.Spec.ExtraLabelNs, ",")))
		}

		if len(n.ins.Spec.ResourceLabels) != 0 {
			args = append(args, fmt.Sprintf("--resource-labels=%s", strings.Join(n.ins.Spec.ResourceLabels, ",")))
		}

		if strings.TrimSpace(n.ins.Spec.LabelWhiteList) != "" {
			args = append(args, fmt.Sprintf("--label-whitelist=%s", n.ins.Spec.LabelWhiteList))
		}

		obj.Spec.Template.Spec.Containers[0].Args = args
	}

	// The Namespace should already be defined, so let's set the namespace
	// to the namespace defined in the DaemonSet object
	obj.SetNamespace(n.ins.GetNamespace())

	// found states if the ConfigMap was found
	found := &appsv1.DaemonSet{}
	r.Log.Info("Looking for DaemonSet '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the DaemonSet to see if it exists, and if so, check if it's
	// Ready/NotReady. If the DaemonSet does not exist, then attempt to
	// create it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	// If we found the DaemonSet, let's attempt to update it
	r.Log.Info("Found, updating")
	err = n.rec.Client.Update(context.TODO(), &obj)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// Service attempts to create a Service in a given Namespace. If the
// Service already exists, then attempt to update it.
func Service(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// Service object, so let's get the resource's Service object
	obj := n.resources[state].Service

	// The Namespace should already be defined, so let's set the
	// namespace to the namespace defined in the ConfigMap object
	obj.SetNamespace(n.ins.GetNamespace())

	// update ports
	if n.ins.Spec.Operand.ServicePort != 0 {
		obj.Spec.Ports[0].Port = int32(n.ins.Spec.Operand.ServicePort)
		obj.Spec.Ports[0].TargetPort = intstr.FromInt(n.ins.Spec.Operand.ServicePort)
	} else {
		obj.Spec.Ports[0].Port = int32(defaultServicePort)
		obj.Spec.Ports[0].TargetPort = intstr.FromInt(defaultServicePort)
	}

	// found states if the DaemonSet was found
	found := &corev1.Service{}
	r.Log.Info("Looking for Service '", obj.Name, "' in Namespace '", obj.Namespace, "'")

	// SetControllerReference sets the owner as a Controller OwnerReference
	// and is used for garbage collection of the controlled object. It is
	// also used to reconcile the owner object on changes to the controlled
	// object. If we cannot set the owner, then return NotReady
	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.Scheme); err != nil {
		return NotReady, err
	}

	// Look for the Service to see if it exists, and if so, check if it's
	// Ready/NotReady. If the Service does not exist, then attempt to create
	// it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	r.Log.Info("Found, updating")

	// Copy the Service object
	required := obj.DeepCopy()

	// Set the resource version based on what we found when
	// searching for the existing Service. Do the same for
	// ClusterIP
	required.ResourceVersion = found.ResourceVersion
	required.Spec.ClusterIP = found.Spec.ClusterIP

	// If we found the DaemonSet, let's attempt to update it
	// with the resource version and cluster IP that we
	// found
	err = n.rec.Client.Update(context.TODO(), required)

	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}

// SecurityContextConstraints attempts to create SecurityContextConstraints
// in a given Namespace. If the scc already exists, then attempt to update it.
func SecurityContextConstraints(n NFD) (ResourceStatus, error) {
	// state represents the resource's 'control' function index
	state := n.idx

	// It is assumed that the index has already been verified to be a
	// scc object, so let's get the resource's scc object
	obj := n.resources[state].SecurityContextConstraints

	// Set the correct namespace for SCC when installed in non default namespace
	obj.Users[0] = "system:serviceaccount:" + n.ins.GetNamespace() + ":" + obj.GetName()

	// found states if the scc was found
	found := &secv1.SecurityContextConstraints{}
	r.Log.Info("Looking for SecurityContextConstraints '", obj.Name, "' in Namespace 'default'")

	// Look for the scc to see if it exists, and if so, check if it's
	// Ready/NotReady. If the scc does not exist, then attempt to create
	// it
	err := n.rec.Client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && apierrors.IsNotFound(err) {
		r.Log.Info("Not found, creating")
		err = n.rec.Client.Create(context.TODO(), &obj)
		if err != nil {
			r.Log.Info("Couldn't create", "Error", err)
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	r.Log.Info("Found, updating")

	// If we found the scc, let's attempt to update it with the resource
	// version we found
	required := obj.DeepCopy()
	required.ResourceVersion = found.ResourceVersion

	// If we found the scc, let's attempt to update it with the resource
	// version we found
	err = n.rec.Client.Update(context.TODO(), required)
	if err != nil {
		return NotReady, err
	}

	return Ready, nil
}
