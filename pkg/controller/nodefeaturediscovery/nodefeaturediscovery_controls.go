package nodefeaturediscovery

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func ServiceAccount(n NFD) error {

	state := n.idx
	obj := &n.resources[state].ServiceAccount

	found := &corev1.ServiceAccount{}
	logger := log.WithValues("ServiceAccount", obj.Namespace, "Namespace", obj.Name)

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	logger.Info("Found")

	return nil
}
func ClusterRole(n NFD) error {

	state := n.idx
	obj := &n.resources[state].ClusterRole

	found := &rbacv1.ClusterRole{}
	logger := log.WithValues("ClusterRole", obj.Namespace, "Namespace", obj.Name)

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	logger.Info("Found")

	return nil
}

func ClusterRoleBinding(n NFD) error {

	state := n.idx
	obj := &n.resources[state].ClusterRoleBinding

	found := &rbacv1.ClusterRoleBinding{}
	logger := log.WithValues("ClusterRoleBinding", obj.Namespace, "Namespace", obj.Name)

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	logger.Info("Found")

	return nil
}

func ConfigMap(n NFD) error {

	state := n.idx
	obj := &n.resources[state].ConfigMap

	found := &corev1.ConfigMap{}
	logger := log.WithValues("ConfigMap", obj.Namespace, "Namespace", obj.Name)

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	logger.Info("Found")

	return nil
}

func DaemonSet(n NFD) error {

	state := n.idx
	obj := &n.resources[state].DaemonSet

	found := &appsv1.DaemonSet{}
	logger := log.WithValues("DaemonSet", obj.Namespace, "Namespace", obj.Name)

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	logger.Info("Found")

	return nil
}

func Service(n NFD) error {

	state := n.idx
	obj := &n.resources[state].Service

	found := &appsv1.DaemonSet{}
	logger := log.WithValues("Service", obj.Namespace, "Namespace", obj.Name)

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	logger.Info("Found")

	return nil
}
