package nodefeaturediscovery

import (
	"context"
	"log"

	nodefeaturediscoveryv1alpha1 "github.com/openshift/cluster-nfd-operator/pkg/apis/nodefeaturediscovery/v1alpha1"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	//	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

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

var nfdsa = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: node-feature-discovery
  namespace: openshift-cluster-node-tuning-operator
`

func nfdServiceAccount(r *ReconcileNodeFeatureDiscovery, nfd *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) *corev1.ServiceAccount {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(nfdsa), nil, nil)
	if err != nil {
		log.Printf("Error decoding ServiceAccount manifest")
		return nil
	}

	err = controllerutil.SetControllerReference(nfd, obj.(*corev1.ServiceAccount), r.scheme)
	if err != nil {
		log.Printf("Couldn't set owner references for ServiceAccount: %v", err)
		return nil
	}
	return obj.(*corev1.ServiceAccount)
}

// Reconcile reads that state of the cluster for a NodeFeatureDiscovery object and makes changes based on the state read
// and what is in the NodeFeatureDiscovery.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNodeFeatureDiscovery) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling NodeFeatureDiscovery %s/%s\n", request.Namespace, request.Name)

	// Fetch the NodeFeatureDiscovery instance
	nfdInstance := &nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery{}
	err := r.client.Get(context.TODO(), request.NamespacedName, nfdInstance)
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

	sa := nfdServiceAccount(r, nfdInstance)

	found := &corev1.ServiceAccount{}
	log.Printf("Lookgin for ServiceAccount:%s in Namespace:%s\n", sa.Name, sa.Namespace)
	err = r.client.Get(context.TODO(), types.NamespacedName{Namespace: sa.Namespace, Name: sa.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating ServiceAccount:%s in Namespace:%s\n", sa.Name, sa.Namespace)
		err = r.client.Create(context.TODO(), sa)
		if err != nil {
			log.Printf("Couldn't create  ServiceAccount:%s in Namespace:%s\n", sa.Name, sa.Namespace)
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// // Define a new Pod object
	// pod := newPodForCR(nfdInstance)

	// // Set NodeFeatureDiscovery instance as the owner and controller
	// if err := controllerutil.SetControllerReference(nfdInstance, pod, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	// }

	// // Check if this Pod already exists
	// found := &corev1.Pod{}

	// err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	// if err != nil && errors.IsNotFound(err) {
	// 	log.Printf("Creating a new Pod %s/%s\n", pod.Namespace, pod.Name)
	// 	err = r.client.Create(context.TODO(), pod)
	// 	if err != nil {
	// 		return reconcile.Result{}, err
	// 	}

	// 	// Pod created successfully - don't requeue
	// 	return reconcile.Result{}, nil
	// } else if err != nil {
	// 	return reconcile.Result{}, err
	// }

	// // Pod already exists - don't requeue
	// log.Printf("Skip reconcile: Pod %s/%s already exists", found.Namespace, found.Name)
	return reconcile.Result{}, nil
}

var deployment = `
apiVersion: v1
kind: Pod
metadata:
  name: cuda-vector-add
  namespace: nvidia
spec:
  restartPolicy: OnFailure
  containers:
    - name: cuda-vector-add
      image: "docker.io/mirrorgooglecontainers/cuda-vector-add:v0.1"
      env:
        - name: NVIDIA_VISIBLE_DEVICES
          value: all
        - name: NVIDIA_DRIVER_CAPABILITIES
          value: "compute,utility"
        - name: NVIDIA_REQUIRE_CUDA
          value: "cuda>=5.0"
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop: ["ALL"]
        seLinuxOptions:
          type: nvidia_container_t

      resources:
        limits:
          nvidia.com/gpu: 1 # requesting 1 GPU
`

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) *corev1.Pod {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, _, err := decode([]byte(deployment), nil, nil)
	if err != nil {
		log.Printf("Error decoding pod manifest")
	}
	return obj.(*corev1.Pod)
}
