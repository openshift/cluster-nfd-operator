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

	"github.com/go-logr/logr"
	security "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	criapierrors "k8s.io/cri-api/pkg/errors"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
)

var log = logf.Log.WithName("controller_nodefeaturediscovery")

var nfd NFD

// NodeFeatureDiscoveryReconciler reconciles a NodeFeatureDiscovery object
type NodeFeatureDiscoveryReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	AssetsDir string
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeFeatureDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// we want to initate reconcile loop only on spec change of the object
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if validateUpdateEvent(&e) {
				return false
			}
			return true
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nfdv1.NodeFeatureDiscovery{}).
		Owns(&appsv1.DaemonSet{}, builder.WithPredicates(p)).
		Owns(&corev1.Service{}, builder.WithPredicates(p)).
		//service acct, rolebinding, role, service, daemonset
		Owns(&corev1.ServiceAccount{}, builder.WithPredicates(p)).
		Owns(&corev1.ServiceAccount{}, builder.WithPredicates(p)).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(p)).
		Owns(&security.SecurityContextConstraints{}).
		Complete(r)
}

func validateUpdateEvent(e *event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		klog.Error("Update event has no old runtime object to update")
		return false
	}
	if e.ObjectNew == nil {
		klog.Error("Update event has no new runtime object for update")
		return false
	}

	return true
}

// +kubebuilder:rbac:groups=nfd.openshift.io,resources=nodefeaturediscoveries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nfd.openshift.io,resources=nodefeaturediscoveries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nfd.openshift.io,resources=nodefeaturediscoveries/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get
// +kubebuilder:rbac:groups=config.openshift.io,resources=proxies,verbs=get;list
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=use;get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=imagestreams/layers,verbs=get
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;update;
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusteroperators,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=clusteroperators/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=issuers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificates,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NodeFeatureDiscoveryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("nodefeaturediscovery", req.NamespacedName)

	// Fetch the NodeFeatureDiscovery instance
	r.Log.Info("Fetch the NodeFeatureDiscovery instance")
	instance := &nfdv1.NodeFeatureDiscovery{}
	err := r.Get(ctx, req.NamespacedName, instance)
	// Error reading the object - requeue the request.
	if err != nil {
		// handle deletion of resource
		if criapierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Info("resource has been deleted", "req", req.Name, "got", instance.Name)
			return ctrl.Result{Requeue: false}, nil
		}

		r.Log.Error(err, "requeueing event since there was an error reading object")
		return ctrl.Result{Requeue: true}, err
	}

	// apply components
	r.Log.Info("Ready to apply components")
	nfd.init(r, instance)
	result, err := applyComponents()

	// If the components could not be applied, then check for degraded conditions
	if err != nil {
		conditions := r.getDegradedConditions("testing", err.Error())
		if err := r.updateStatus(instance, conditions); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	///XXX TODO: Reorder these and add "updateAvailableConditions" for each
	// Check the status of the NFD Operator Service
	rstatus, err := r.getServiceConditions(instance, ctx)
	if rstatus.isDegraded == true {
		r.Log.Info("Failed getting NFD operator Service")
		return r.updateDegradedCondition(instance, err.Error(), err)
	}

	// Check the status of the NFD Operator Worker DaemonSet
	rstatus, err = r.getWorkerDaemonSetConditions(instance, ctx)
	if rstatus.isProgressing == true {
		r.Log.Info("NFD operator Worker DaemonSet is progressing.")
		return r.updateProgressingCondition(instance, "NFDOperatorWorkerDaemonSetIsProgressing", err)

	} else if rstatus.isDegraded == true {
		r.Log.Info("Failed getting NFD operator DaemonSet")
		return r.updateDegradedCondition(instance, err.Error(), err)

	} else if err != nil {
		r.Log.Info("Unknown error when trying to verify NFD Operator Daemon Set.")
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDWorkerDaemonSet, err)
	}

	/*
	// Check the status of the NFD Operator cluster roles
	rstatus, err = r.getRoleConditions(instance)
	if rstatus.isProgressing == true {
		r.Log.Info("NFD operator cluster role is progressing.")
		return r.updateProgressingCondition(instance, conditionNFDRoleDegraded, nil)

	} else if rstatus.isDegraded == true {
		r.Log.Info("Failed getting NFD operator Role")
		return r.updateDegradedCondition(instance, conditionNFDRoleDegraded, err)

	} else if err != nil {
		r.Log.Info("Unknown error when trying to verify NFD Operator cluster role.")
		return r.updateDegradedCondition(instance, conditionNFDRoleDegraded, err)
	}

	// Check the status of the NFD Operator worker ConfigMap
	rstatus, err = r.getWorkerConfigConditions(instance)
	if rstatus.isProgressing == true {
		r.Log.Info("NFD operator worker config is progressing.")
		return r.updateProgressingCondition(instance, conditionNFDWorkerConfigDegraded, nil)

	} else if rstatus.isDegraded == true {
		r.Log.Error(err, "Failed getting NFD Operator worker Config Map")
		return r.updateDegradedCondition(instance, conditionNFDWorkerConfigDegraded, err)

	} else if err != nil {
		r.Log.Info("Unknown error when trying to verify NFD Operator worker Config Map.")
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDWorkerConfig, err)
	}

	// Check the status of the NFD Operator role binding
	rstatus, err = r.getRoleBindingConditions(instance)
	if rstatus.isProgressing == true {
		r.Log.Info("NFD operator cluster role binding is progressing.")
		//return r.updateProgressingCondition(instance, conditionNFDClusterRoleDegraded, nil)

	} else if rstatus.isDegraded == true {
		r.Log.Error(err, "Failed getting NFD operator Role Binding")
		return r.updateDegradedCondition(instance, conditionNFDRoleBindingDegraded, err)

	} else if err != nil {
		r.Log.Info("Unknown error when trying to verify NFD Operator Role Binding.")
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDRoleBinding, err)
	}

	// Check the status of the NFD Operator ServiceAccount
	rstatus, err = r.getServiceAccountConditions(instance)
	if rstatus.isProgressing == true {
		r.Log.Info("NFD operator service account is progressing.")
		return r.updateProgressingCondition(instance, conditionNFDServiceAccountDegraded, nil)

	} else if rstatus.isDegraded == true {
		r.Log.Error(err, "Failed getting NFD operator Service Account")
		return r.updateDegradedCondition(instance, conditionNFDServiceAccountDegraded, err)

	} else if err != nil {
		r.Log.Info("Unknown error when trying to verify NFD Operator Service Account.")
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDServiceAccount, err)
	}
	*/

	// Get available conditions
	conditions := r.getAvailableConditions()

	// Update the status of the resource on the CRD
	r.updateStatus(instance, conditions)

	//return r.updateAvailableCondition(instance)

	if err := r.updateStatus(instance, conditions); err != nil {
		if &result != nil {
			return result, nil
		}
		return reconcile.Result{}, err
	}

	if &result != nil {
		return result, nil
	}

	return ctrl.Result{}, nil
}

func applyComponents() (reconcile.Result, error) {

	for {
		err := nfd.step()
		if err != nil {
			return reconcile.Result{}, err
		}
		if nfd.last() {
			break
		}
	}
	return ctrl.Result{}, nil
}
