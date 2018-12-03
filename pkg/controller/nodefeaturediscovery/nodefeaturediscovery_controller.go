package nodefeaturediscovery

import (
	"context"

	nodefeaturediscoveryv1alpha1 "github.com/openshift/cluster-nfd-operator/pkg/apis/nodefeaturediscovery/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var logc = logf.Log.WithName("controller_nfd")

// Add creates a new NodeFeatureDiscovery Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nodefeaturediscovery-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NodeFeatureDiscovery
	err = c.Watch(&source.Kind{Type: &nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner NodeFeatureDiscovery
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery{},
	})
	if err != nil {
		return err
	}

	return nil
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNodeFeatureDiscovery{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
}

var _ reconcile.Reconciler = &ReconcileNodeFeatureDiscovery{}

// ReconcileNodeFeatureDiscovery reconciles a NodeFeatureDiscovery object
type ReconcileNodeFeatureDiscovery struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}


// Reconcile reads that state of the cluster for a NodeFeatureDiscovery object and makes changes based on the state read
// and what is in the NodeFeatureDiscovery.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNodeFeatureDiscovery) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	//log.Printf("Reconciling NodeFeatureDiscovery %s/%s\n", request.Namespace, request.Name)

	reqLogger := logc.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling NodeFeatureDiscovery.")
	// Fetch the NodeFeatureDiscovery instance
	ins := &nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery{}
	err := r.client.Get(context.TODO(), request.NamespacedName, ins)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	err = setOwnerReferenceForAll(r, ins)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = serviceAccountControl(r, ins)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = clusterRoleControl(r, ins)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = clusterRoleBindingControl(r, ins)
	if err != nil {
		return reconcile.Result{}, err
	}

	 // err = securityContextConstraintControl(r, ins)
	 // if err != nil {
	 // 	 return reconcile.Result{}, err
	 // }

	err = configMapControl(r, ins)
	if err != nil {
		return reconcile.Result{}, err
	}
	
	err = daemonSetControl(r, ins)
	if err != nil {
		return reconcile.Result{}, err
	}
	
	return reconcile.Result{}, nil
}
