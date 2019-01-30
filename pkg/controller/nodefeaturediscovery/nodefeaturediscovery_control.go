package nodefeaturediscovery

import (
	"context"

	securityv1 "github.com/openshift/api/security/v1"
	nodefeaturediscoveryv1alpha1 "github.com/openshift/cluster-nfd-operator/pkg/apis/nfd/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type controlFunc func(*ReconcileNodeFeatureDiscovery, *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error

var nfdControl []controlFunc

func init() {
	nfdControl = append(nfdControl, setOwnerReferenceForAll)
	nfdControl = append(nfdControl, serviceAccountControl)
	nfdControl = append(nfdControl, clusterRoleControl)
	nfdControl = append(nfdControl, clusterRoleBindingControl)
	nfdControl = append(nfdControl, configMapControl)
	//	nfdControl = append(nfdControl, securityContextConstraintControl)
	nfdControl = append(nfdControl, daemonSetControl)
}

func setOwnerReferenceForAll(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	err := controllerutil.SetControllerReference(ins, &nfdServiceAccount, r.scheme)
	if err != nil {
		log.Info("Couldn't set owner references for ServiceAccount: ", err)
		return err
	}
	err = controllerutil.SetControllerReference(ins, &nfdClusterRole, r.scheme)
	if err != nil {
		log.Info("Couldn't set owner references for ClusterRole: ", err)
		return err
	}
	err = controllerutil.SetControllerReference(ins, &nfdClusterRoleBinding, r.scheme)
	if err != nil {
		log.Info("Couldn't set owner references for ClusterRoleBinding: ", err)
		return err
	}
	err = controllerutil.SetControllerReference(ins, &nfdSecurityContextConstraint, r.scheme)
	if err != nil {
		log.Info("Couldn't set owner references for SecurityContextConstraint: ", err)
		return err
	}
	err = controllerutil.SetControllerReference(ins, &nfdDaemonSet, r.scheme)
	if err != nil {
		log.Info("Couldn't set owner references for DaemonSet: ", err)
		return err
	}

	return nil
}

func serviceAccountControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdServiceAccount
	found := &corev1.ServiceAccount{}

	log.Info("Looking for ServiceAccount:%s in Namespace:%s\n", obj.Name, obj.Namespace)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Not found creating ServiceAccount:%s in Namespace:%s\n", obj.Name, obj.Namespace)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Info("Couldn't create Namespace:%s\n%v\n", obj.Name, err)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Info("Found ServiceAccount:%s in Namespace:%s\n", obj.Name, obj.Namespace)

	return nil
}

func clusterRoleControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdClusterRole
	found := &rbacv1.ClusterRole{}

	log.Info("Looking for ClusterRole:%s\n", obj.Name)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Not found creating ClusterRole:%s\n", obj.Name)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Info("Couldn't create ClusterRole:%s\n%v\n", obj.Name, err)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Info("Found ClusterRole:%s\n", obj.Name)

	return nil
}

func clusterRoleBindingControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdClusterRoleBinding
	found := &rbacv1.ClusterRoleBinding{}

	log.Info("Looking for ClusterRoleBinding:%s\n", obj.Name)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Not found creating ClusterRoleBinding:%s\n", obj.Name)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Info("Couldn't create ClusterRoleBinding:%s\n%v\n", obj.Name, err)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Info("Found ClusterRoleBinding:%s\n", obj.Name)

	return nil
}

func configMapControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdConfigMap
	found := &corev1.ConfigMap{}

	log.Info("Looking for ConfigMap:%s in Namespace:%s\n", obj.Name, obj.Namespace)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Not found creating ConfigMap:%s in Namespace:%s\n", obj.Name, obj.Namespace)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Info("Couldn't create ConfigMap:%s in Namespace:%s\n%v\n", obj.Name, obj.Namespace, err)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Info("Found ConfigMap:%s\n", obj.Name)

	return nil
}

func daemonSetControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdDaemonSet
	found := &appsv1.DaemonSet{}

	log.Info("Looking for DaemonSet:%s in Namespace:%s\n", obj.Name, obj.Namespace)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Not found creating DaemonSet:%s in Namespace:%s\n", obj.Name, obj.Namespace)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Info("Couldn't create DaemonSet:%s in Namespace:%s\n%v\n", obj.Name, obj.Namespace, err)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Info("Found DaemonSet:%s\n", obj.Name)

	return nil
}

func securityContextConstraintControl(r *ReconcileNodeFeatureDiscovery,
	ins *nodefeaturediscoveryv1alpha1.NodeFeatureDiscovery) error {

	obj := &nfdSecurityContextConstraint
	found := &securityv1.SecurityContextConstraints{}

	log.Info("Looking for SecurityContextConstraint:%s\n", obj.Name)
	err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Not found creating SecurityContextConstraint:%s\n", obj.Name)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			log.Info("Couldn't create SecurityContextConstraint:%s\n", obj.Name)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	log.Info("Found SecurityContextConstraint:%s\n", obj.Name)

	return nil
}
