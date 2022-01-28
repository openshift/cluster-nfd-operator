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

	security "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	nfdMetrics "github.com/openshift/cluster-nfd-operator/pkg/metrics"
)

var nfd NFD

const finalizer = "foreground-deletion"

// NodeFeatureDiscoveryLogger is a dummy logger struct that is used with
// the NodeFeatureDiscoveryReconciler to initiate a logger.
type NodeFeatureDiscoveryLogger struct {
}

func (log *NodeFeatureDiscoveryLogger) Info(args ...interface{}) {
	klog.Info(args)
}

func (log *NodeFeatureDiscoveryLogger) Infof(format string, args ...interface{}) {
	klog.Infof(format, args)
}

func (log *NodeFeatureDiscoveryLogger) Error(args ...interface{}) {
	klog.Error(args)
}

// NodeFeatureDiscoveryReconciler reconciles a NodeFeatureDiscovery object.
// Below is a description of each field within this struct:
//
//	- client.Client reads and writes directly from/to the OCP API server.
//	  This field needs to be added to the reconciler because it is
//	  responsible for for fetching objects from the server, which NFD
//	  needs to do in order to add its labels to each node in the cluster.
//
//	- Log is used to log the reconciliation. Every controllers needs this.
//
//	- Scheme is used by the kubebuilder library to set OwnerReferences.
//	  Every controller needs this.
//
//	- Recorder defines interfaces for working with OCP event recorders.
//	  This field is needed by NFD in order for NFD to write events.
//
//	- AssetsDir defines the directory with assets under the operator image
type NodeFeatureDiscoveryReconciler struct {
	client.Client
	Log       NodeFeatureDiscoveryLogger
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	AssetsDir string
}

// SetupWithManager sets up the controller with the Manager in order to create
// the controller. The Manager serves the purpose of initializing shared
// dependencies (like caches and clients) from the 'client.Client' field in the
// NodeFeatureDiscoveryReconciler struct.
func (r *NodeFeatureDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// For handling the the creation, deletion, and updates of DaemonSet objects
	dsPredicateFuncs := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			// Extract the old and new DaemonSet objects. If either one
			// doesn't exist, then no update occurred, so return 'false'.
			oldDsObject, ok := e.ObjectOld.(*appsv1.DaemonSet)
			if !ok {
				return false
			}

			newDsObject, ok := e.ObjectNew.(*appsv1.DaemonSet)
			if !ok {
				return false
			}

			// Get the deletion timestamps. If they're the same, then no update
			// has been made.
			oldDeletionTimestamp := oldDsObject.GetDeletionTimestamp()
			newDeletionTimestamp := newDsObject.GetDeletionTimestamp()
			if oldDeletionTimestamp == newDeletionTimestamp {
				return false
			}

			// If everything else is the same, then no update has been made
			// either.
			if newDsObject.GetGeneration() == oldDsObject.GetGeneration() {
				return false
			}

			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {

			// Evaluates to false if the object has been deleted
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {

			// Check if the DaemonSet object has been created already.
			_, ok := e.Object.(*appsv1.DaemonSet)
			return ok
		},
	}

	// For handling the the creation, deletion, and updates of ServiceAccount objects
	saPredicateFuncs := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			// Extract the old and new ServiceAccount objects. If either one
			// doesn't exist, then no update occurred, so return 'false'.
			oldSaObject, ok := e.ObjectOld.(*corev1.ServiceAccount)
			if !ok {
				return false
			}

			newSaObject, ok := e.ObjectNew.(*corev1.ServiceAccount)
			if !ok {
				return false
			}

			// Get the deletion timestamps. If they're the same, then no update
			// has been made.
			oldDeletionTimestamp := oldSaObject.GetDeletionTimestamp()
			newDeletionTimestamp := newSaObject.GetDeletionTimestamp()
			if oldDeletionTimestamp == newDeletionTimestamp {
				return false
			}

			// If everything else is the same, then no update has been made
			// either.
			if newSaObject.GetGeneration() == oldSaObject.GetGeneration() {
				return false
			}

			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {

			// Evaluates to false if the object has been deleted
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {

			// Check if the ServiceAccount object has been created already.
			_, ok := e.Object.(*corev1.ServiceAccount)
			return ok
		},
	}

	// For handling the the creation, deletion, and updates of Service objects
	svcPredicateFuncs := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			// Extract the old and new Service objects. If either one doesn't
			// exist, then no update occurred, so return 'false'.
			oldSvcObject, ok := e.ObjectOld.(*corev1.Service)
			if !ok {
				return false
			}

			newSvcObject, ok := e.ObjectNew.(*corev1.Service)
			if !ok {
				return false
			}

			// Get the deletion timestamps. If they're the same, then no update
			// has been made.
			oldDeletionTimestamp := oldSvcObject.GetDeletionTimestamp()
			newDeletionTimestamp := newSvcObject.GetDeletionTimestamp()
			if oldDeletionTimestamp == newDeletionTimestamp {
				return false
			}

			// If everything else is the same, then no update has been made
			// either.
			if newSvcObject.GetGeneration() == oldSvcObject.GetGeneration() {
				return false
			}

			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {

			// Evaluates to false if the object has been deleted
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {

			// Check if the Service object has been created already.
			_, ok := e.Object.(*corev1.Service)
			return ok
		},
	}

	// For handling the the creation, deletion, and updates of RoleBinding objects
	rbPredicateFuncs := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			// Extract the old and new RoleBinding objects. If either one
			// doesn't exist, then no update occurred, so return 'false'.
			oldRbObject, ok := e.ObjectOld.(*rbacv1.RoleBinding)
			if !ok {
				return false
			}

			newRbObject, ok := e.ObjectNew.(*rbacv1.RoleBinding)
			if !ok {
				return false
			}

			// Get the deletion timestamps. If they're the same, then no update
			// has been made.
			oldDeletionTimestamp := oldRbObject.GetDeletionTimestamp()
			newDeletionTimestamp := newRbObject.GetDeletionTimestamp()
			if oldDeletionTimestamp == newDeletionTimestamp {
				return false
			}

			// If everything else is the same, then no update has been made
			// either.
			if newRbObject.GetGeneration() == oldRbObject.GetGeneration() {
				return false
			}

			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {

			// Evaluates to false if the object has been deleted
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {

			// Check if the RoleBinding object has been created already.
			_, ok := e.Object.(*rbacv1.RoleBinding)
			return ok
		},
	}

	// For handling the the creation, deletion, and updates of Role objects
	rPredicateFuncs := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			// Extract the old and new Role objects. If either one doesn't
			// exist, then no update occurred, so return false.
			oldRObject, ok := e.ObjectOld.(*rbacv1.Role)
			if !ok {
				return false
			}

			newRObject, ok := e.ObjectNew.(*rbacv1.Role)
			if !ok {
				return false
			}

			// Get the deletion timestamps. If they're the same, then no update
			// has been made.
			oldDeletionTimestamp := oldRObject.GetDeletionTimestamp()
			newDeletionTimestamp := newRObject.GetDeletionTimestamp()
			if oldDeletionTimestamp == newDeletionTimestamp {
				return false
			}

			// If everything else is the same, then no update has been made
			// either.
			if newRObject.GetGeneration() == oldRObject.GetGeneration() {
				return false
			}

			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {

			// Evaluates to false if the object has been deleted
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {

			// Check if the Role object has been created already.
			_, ok := e.Object.(*rbacv1.Role)
			return ok
		},
	}

	// For handling the the creation, deletion, and updates of ConfigMap objects
	cmPredicateFuncs := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			// Extract the old and new ConfigMap objects. If either
			// one doesn't exist, then no update occurred, so return
			// 'false'.
			oldCmObject, ok := e.ObjectOld.(*corev1.ConfigMap)
			if !ok {
				return false
			}

			newCmObject, ok := e.ObjectNew.(*corev1.ConfigMap)
			if !ok {
				return false
			}

			// Get the deletion timestamps. If they're the same, then no update
			// has been made.
			oldDeletionTimestamp := oldCmObject.GetDeletionTimestamp()
			newDeletionTimestamp := newCmObject.GetDeletionTimestamp()
			if oldDeletionTimestamp == newDeletionTimestamp {
				return false
			}

			// If everything else is the same, then no update has been made
			// either.
			if newCmObject.GetGeneration() == oldCmObject.GetGeneration() {
				return false
			}

			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {

			// Evaluates to false if the object has been deleted
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {

			// Check if the ConfigMap object has been created already.
			_, ok := e.Object.(*corev1.ConfigMap)
			return ok
		},
	}

	// For handling the the creation, deletion, and updates of SecurityContextConstraints
	// objects
	sccPredicateFuncs := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			// Extract the old and new SecurityContextConstraints objects. If
			// either one doesn't exist, then no update occurred, return 'false'.
			oldSccObject, ok := e.ObjectOld.(*security.SecurityContextConstraints)
			if !ok {
				return false
			}

			newSccObject, ok := e.ObjectNew.(*security.SecurityContextConstraints)
			if !ok {
				return false
			}

			// Get the deletion timestamps. If they're the same, then no update
			// has been made.
			oldDeletionTimestamp := oldSccObject.GetDeletionTimestamp()
			newDeletionTimestamp := newSccObject.GetDeletionTimestamp()
			if oldDeletionTimestamp == newDeletionTimestamp {
				return false
			}

			// If everything else is the same, then no update has been made
			// either.
			if newSccObject.GetGeneration() == oldSccObject.GetGeneration() {
				return false
			}

			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {

			// Evaluates to false if the object has been deleted
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {

			// Check if the SecurityContextConstraints object has been
			// created already.
			_, ok := e.Object.(*security.SecurityContextConstraints)
			return ok
		},
	}

	// For handling the the creation, deletion, and updates of NFD instances
	nfdPredicateFuncs := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			// Extract the old and new NodeFeatureDiscovery instances. If
			// either one doesn't exist, then no update occurred, return 'false'.
			oldNfdObject, ok := e.ObjectOld.(*nfdv1.NodeFeatureDiscovery)
			if !ok {
				return false
			}

			newNfdObject, ok := e.ObjectNew.(*nfdv1.NodeFeatureDiscovery)
			if !ok {
				return false
			}

			// If everything else is the same, then no update has been made
			// either.
			return oldNfdObject.GetGeneration() != newNfdObject.GetGeneration() ||
				!apiequality.Semantic.DeepEqual(oldNfdObject.GetLabels(), newNfdObject.GetLabels())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {

			// Evaluates to false if the object has been deleted
			return !e.DeleteStateUnknown
		},
		CreateFunc: func(e event.CreateEvent) bool {

			// Check if the NodeFeatureDiscovery instance has been created
			// already.
			_, ok := e.Object.(*nfdv1.NodeFeatureDiscovery)
			return ok
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&nfdv1.NodeFeatureDiscovery{}, builder.WithPredicates(nfdPredicateFuncs)).
		Owns(&corev1.ServiceAccount{}, builder.WithPredicates(saPredicateFuncs)).
		Owns(&rbacv1.RoleBinding{}, builder.WithPredicates(rbPredicateFuncs)).
		Owns(&rbacv1.Role{}, builder.WithPredicates(rPredicateFuncs)).
		Owns(&corev1.Service{}, builder.WithPredicates(svcPredicateFuncs)).
		Owns(&appsv1.DaemonSet{}, builder.WithPredicates(dsPredicateFuncs)).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(cmPredicateFuncs)).
		Owns(&security.SecurityContextConstraints{}, builder.WithPredicates(sccPredicateFuncs)).
		Complete(r)
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
	// Fetch the NodeFeatureDiscovery instance
	r.Log.Info("Fetch the NodeFeatureDiscovery instance")
	instance := &nfdv1.NodeFeatureDiscovery{}
	err := r.Get(ctx, req.NamespacedName, instance)
	// Error reading the object - requeue the request.
	if err != nil {

		// handle deletion of resource
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			r.Log.Infof("resource has been deleted", "req", req.Name, "got", instance.Name)
			return ctrl.Result{Requeue: false}, nil
		}

		r.Log.Error(err, "requeueing event since there was an error reading object")
		return ctrl.Result{Requeue: true}, err
	}

	// If the resources are to be deleted, first check to see if the
	// deletion timestamp pointer is not nil. A non-nil value indicates
	// someone or something has triggered the deletion.
	if instance.DeletionTimestamp != nil {
		return r.finalizeNFDOperator(ctx, instance, finalizer)
	}

	// If the finalizer doesn't exist, add it.
	if !r.hasFinalizer(instance, finalizer) {
		return r.addFinalizer(ctx, instance, finalizer)
	}

	// Register NFD instance metrics
	if instance.Spec.Instance != "" {
		nfdMetrics.RegisterInstance(instance.Spec.Instance, instance.ObjectMeta.Namespace)
	}

	// apply components
	r.Log.Info("Ready to apply components")
	nfd.init(r, instance)
	result, err := applyComponents()

	// If the components could not be applied, then check for degraded conditions
	if err != nil {
		nfdMetrics.Degraded(true)
		conditions := r.getDegradedConditions("Degraded", err.Error())
		if err := r.updateStatus(instance, conditions); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, err
	}

	// Check the status of the NFD Operator Worker ServiceAccount
	if rstatus, err := r.getWorkerServiceAccountConditions(ctx, instance); err != nil {
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDWorkerServiceAccount, err.Error())
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, conditionNFDWorkerServiceAccountDegraded, "nfd-worker service account has been degraded")
	}

	// Check the status of the NFD Operator Master ServiceAccount
	if rstatus, err := r.getMasterServiceAccountConditions(ctx, instance); err != nil {
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDMasterServiceAccount, err.Error())
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, conditionNFDMasterServiceAccountDegraded, "nfd-master service account has been degraded")
	}

	// Check the status of the NFD Operator role
	if rstatus, err := r.getRoleConditions(ctx, instance); err != nil {
		return r.updateDegradedCondition(instance, conditionNFDRoleDegraded, err.Error())
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, conditionNFDRoleDegraded, "nfd-worker role has been degraded")
	}

	// Check the status of the NFD Operator cluster role
	if rstatus, err := r.getClusterRoleConditions(ctx); err != nil {
		return r.updateDegradedCondition(instance, conditionNFDClusterRoleDegraded, err.Error())
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, conditionNFDClusterRoleDegraded, "nfd ClusterRole has been degraded")
	}

	// Check the status of the NFD Operator cluster role binding
	if rstatus, err := r.getClusterRoleBindingConditions(ctx); err != nil {
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDClusterRoleBinding, err.Error())
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, conditionNFDClusterRoleBindingDegraded, "nfd ClusterRoleBinding has been degraded")
	}

	// Check the status of the NFD Operator role binding
	if rstatus, err := r.getRoleBindingConditions(ctx, instance); err != nil {
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDRoleBinding, err.Error())
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, conditionNFDRoleBindingDegraded, "nfd RoleBinding has been degraded")
	}

	// Check the status of the NFD Operator Service
	if rstatus, err := r.getServiceConditions(ctx, instance); err != nil {
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDService, err.Error())
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, conditionNFDServiceDegraded, "nfd Service has been degraded")
	}

	// Check the status of the NFD Operator worker ConfigMap
	if rstatus, err := r.getWorkerConfigConditions(nfd); err != nil {
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDWorkerConfig, err.Error())
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, conditionNFDWorkerConfigDegraded, "nfd-worker ConfigMap has been degraded")
	}

	// Check the status of the NFD Operator Worker DaemonSet
	if rstatus, err := r.getWorkerDaemonSetConditions(ctx, instance); err != nil {
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDWorkerDaemonSet, err.Error())
	} else if rstatus.isProgressing {
		return r.updateProgressingCondition(instance, err.Error(), "nfd-worker Daemonset is progressing")
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, err.Error(), "nfd-worker Daemonset has been degraded")
	}

	// Check the status of the NFD Operator Master DaemonSet
	if rstatus, err := r.getMasterDaemonSetConditions(ctx, instance); err != nil {
		return r.updateDegradedCondition(instance, conditionFailedGettingNFDMasterDaemonSet, err.Error())
	} else if rstatus.isProgressing {
		return r.updateProgressingCondition(instance, err.Error(), "nfd-master Daemonset is progressing")
	} else if rstatus.isDegraded {
		return r.updateDegradedCondition(instance, err.Error(), "nfd-master Daemonset has been degraded")
	}

	// Get available conditions
	conditions := r.getAvailableConditions()

	// Update the status of the resource on the CRD
	if err := r.updateStatus(instance, conditions); err != nil {
		if result != nil {
			return *result, nil
		}
		return reconcile.Result{}, err
	}

	if result != nil {
		return *result, nil
	}

	// All objects are healthy during reconcile loop
	return ctrl.Result{}, nil
}

func applyComponents() (*reconcile.Result, error) {
	for {
		err := nfd.step()
		if err != nil {
			return &reconcile.Result{}, err
		}
		if nfd.last() {
			break
		}
	}
	return &ctrl.Result{}, nil
}
