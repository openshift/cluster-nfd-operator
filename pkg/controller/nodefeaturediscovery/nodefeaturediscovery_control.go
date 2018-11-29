package nodefeaturediscovery

import (
	"context"
	"log"

	nodefeaturediscoveryv1alpha1 "github.com/openshift/cluster-nfd-operator/pkg/apis/nodefeaturediscovery/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	securityv1 "github.com/openshift/api/security/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"k8s.io/apimachinery/pkg/types"
)


func setOwnerReferenceForAll(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	err := controllerutil.SetControllerReference(ins, &nfdServiceAccount, r.scheme)
	if err != nil {
		log.Printf("Couldn't set owner references for ServiceAccount: %v", err)
		return err
	}
	err = controllerutil.SetControllerReference(ins, &nfdClusterRole, r.scheme)
	if err != nil {
	 	log.Printf("Couldn't set owner references for ClusterRole: %v", err)
	 	return err
	}
	err = controllerutil.SetControllerReference(ins, &nfdClusterRoleBinding, r.scheme)
	if err != nil {
	 	log.Printf("Couldn't set owner references for ClusterRoleBinding: %v", err)
	 	return err
	}
	err = controllerutil.SetControllerReference(ins, &nfdSecurityContextConstraint, r.scheme)
	if err != nil {
	 	log.Printf("Couldn't set owner references for SecurityContextConstraint: %v", err)
	 	return err
	}
	err = controllerutil.SetControllerReference(ins, &nfdDaemonSet, r.scheme)
	if err != nil {
	 	log.Printf("Couldn't set owner references for DaemonSet: %v", err)
	 	return err
	}
	
	return nil
}

func serviceAccountControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdServiceAccount 
	found := &corev1.ServiceAccount{}
	
	log.Printf("Looking for ServiceAccount:%s in Namespace:%s\n", obj.Name, obj.Namespace)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating ServiceAccount:%s in Namespace:%s\n", obj.Name, obj.Namespace)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Printf("Couldn't create Namespace:%s\n", obj.Name)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Printf("Found ServiceAccount:%s in Namespace:%s\n", obj.Name, obj.Namespace)
	
	return nil
}

func clusterRoleControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdClusterRole
	found := &rbacv1.ClusterRole{}
	
	log.Printf("Looking for ClusterRole:%s\n", obj.Name)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating ClusterRole:%s\n", obj.Name)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Printf("Couldn't create ClusterRole:%s\n", obj.Name)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Printf("Found ClusterRole:%s\n", obj.Name )
	
	return nil
}

func clusterRoleBindingControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdClusterRoleBinding
	found := &rbacv1.ClusterRoleBinding{}
	
	log.Printf("Looking for ClusterRoleBinding:%s\n", obj.Name)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating ClusterRoleBinding:%s\n", obj.Name)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Printf("Couldn't create ClusterRoleBinding:%s\n", obj.Name)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Printf("Found ClusterRoleBinding:%s\n", obj.Name )
	
	return nil
}

func daemonSetControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdDaemonSet
	found := &appsv1.DaemonSet{}
	
	log.Printf("Looking for DaemonSet:%s in Namespace:%s\n", obj.Name, obj.Namespace)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating DaemonSet:%s in Namespace:%s\n", obj.Name, obj.Namespace)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Printf("Couldn't create DaemonSet:%s in Namespace:%S\n", obj.Name, obj.Namespace)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Printf("Found DaemonSet:%s\n", obj.Name, obj.Namespace )
	
	return nil
}


func securityContextConstraintControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdSecurityContextConstraint
	found := &securityv1.SecurityContextConstraints{}
	
	log.Printf("Looking for SecurityContextConstraint:%s\n", obj.Name)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Printf("Creating SecurityContextConstraint:%s\n", obj.Name)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Printf("Couldn't create SecurityContextConstraint:%s\n", obj.Name)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Printf("Found SecurityContextConstraint:%s\n", obj.Name )
	
	return nil
}
