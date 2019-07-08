package nodefeaturediscovery

import (
	"context"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type controlFunc []func(n NFD) (ResourceStatus, error)

type ResourceStatus int

const (
	Ready    ResourceStatus = 0
	NotReady ResourceStatus = 1
)

func (s ResourceStatus) String() string {
	names := [...]string{
		"Ready",
		"NotReady"}

	if s < Ready || s > NotReady {
		return "Unkown Resources Status"
	}
	return names[s]
}

func ServiceAccount(n NFD) (ResourceStatus, error) {

	state := n.idx
	obj := n.resources[state].ServiceAccount

	found := &corev1.ServiceAccount{}

	//obj.ObjectMeta.ResourceVersion = ""

	logger := log.WithValues("ServiceAccount", obj.Name, "Namespace", obj.Namespace, "Res", obj.GetObjectMeta().GetResourceVersion())

	logger.Info("XXXX")

	if err := controllerutil.SetControllerReference(n.ins, &obj, n.rec.scheme); err != nil {
		return NotReady, err
	}

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating ", err.Error())
		err = n.rec.client.Create(context.TODO(), &obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found")

	return Ready, nil
}
func ClusterRole(n NFD) (ResourceStatus, error) {

	state := n.idx
	obj := &n.resources[state].ClusterRole

	found := &rbacv1.ClusterRole{}
	logger := log.WithValues("ClusterRole", obj.Name, "Namespace", obj.Namespace)

	if err := controllerutil.SetControllerReference(n.ins, obj, n.rec.scheme); err != nil {
		return NotReady, err
	}

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found")

	return Ready, nil
}

func ClusterRoleBinding(n NFD) (ResourceStatus, error) {

	state := n.idx
	obj := &n.resources[state].ClusterRoleBinding

	found := &rbacv1.ClusterRoleBinding{}
	logger := log.WithValues("ClusterRoleBinding", obj.Name, "Namespace", obj.Namespace)

	if err := controllerutil.SetControllerReference(n.ins, obj, n.rec.scheme); err != nil {
		return NotReady, err
	}

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found")

	return Ready, nil
}
func Role(n NFD) (ResourceStatus, error) {

	state := n.idx
	obj := &n.resources[state].Role

	found := &rbacv1.Role{}
	logger := log.WithValues("Role", obj.Name, "Namespace", obj.Namespace)

	if err := controllerutil.SetControllerReference(n.ins, obj, n.rec.scheme); err != nil {
		return NotReady, err
	}

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found")

	return Ready, nil
}

func RoleBinding(n NFD) (ResourceStatus, error) {

	state := n.idx
	obj := &n.resources[state].RoleBinding

	found := &rbacv1.RoleBinding{}
	logger := log.WithValues("RoleBinding", obj.Name, "Namespace", obj.Namespace)

	if err := controllerutil.SetControllerReference(n.ins, obj, n.rec.scheme); err != nil {
		return NotReady, err
	}

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found")

	return Ready, nil
}

func ConfigMap(n NFD) (ResourceStatus, error) {

	state := n.idx
	obj := &n.resources[state].ConfigMap

	found := &corev1.ConfigMap{}
	logger := log.WithValues("ConfigMap", obj.Name, "Namespace", obj.Namespace)

	if err := controllerutil.SetControllerReference(n.ins, obj, n.rec.scheme); err != nil {
		return NotReady, err
	}

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found")

	return Ready, nil
}

func DaemonSet(n NFD) (ResourceStatus, error) {

	state := n.idx
	obj := &n.resources[state].DaemonSet

	found := &appsv1.DaemonSet{}
	logger := log.WithValues("DaemonSet", obj.Name, "Namespace", obj.Namespace)

	if err := controllerutil.SetControllerReference(n.ins, obj, n.rec.scheme); err != nil {
		return NotReady, err
	}

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found")

	return isDaemonSetReady(obj, n), nil
}

func Service(n NFD) (ResourceStatus, error) {

	state := n.idx
	obj := &n.resources[state].Service

	found := &corev1.Service{}
	logger := log.WithValues("Service", obj.Name, "Namespace", obj.Namespace)

	if err := controllerutil.SetControllerReference(n.ins, obj, n.rec.scheme); err != nil {
		return NotReady, err
	}

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: obj.Namespace, Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create")
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found")

	return Ready, nil
}

func SecurityContextConstraints(n NFD) (ResourceStatus, error) {

	state := n.idx
	obj := &n.resources[state].SecurityContextConstraints

	found := &secv1.SecurityContextConstraints{}
	logger := log.WithValues("SecurityContextConstraints", obj.Name, "Namespace", "default")

	if err := controllerutil.SetControllerReference(n.ins, obj, n.rec.scheme); err != nil {
		return NotReady, err
	}

	logger.Info("Looking for")
	err := n.rec.client.Get(context.TODO(), types.NamespacedName{Namespace: "", Name: obj.Name}, found)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Not found, creating")
		err = n.rec.client.Create(context.TODO(), obj)
		if err != nil {
			logger.Info("Couldn't create", "Error", err)
			return NotReady, err
		}
		return Ready, nil
	} else if err != nil {
		return NotReady, err
	}

	logger.Info("Found")

	return Ready, nil
}
