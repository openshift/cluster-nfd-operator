package nodefeaturediscovery

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"log"
)

var nfdNameSpace *corev1.Namespace
var nfdServiceAccount *corev1.ServiceAccount
var nfdRole *rbacv1.ClusterRole
var nfdRoleBinding *rbacv1.ClusterRoleBinding

//var nfdSCC *
var nfdDaemonSet *appsv1.DaemonSet

func decodeManifest(yaml string) interface{} {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(yaml), nil, nil)
	if err != nil {
		log.Printf("Error decoding ServiceAccount manifest")
		return nil
	}
	return obj
}

func init() {
	nfdNameSpace = decodeManifest(nfdnamespace).(*corev1.Namespace)
	nfdServiceAccount = decodeManifest(nfdserviceaccount).(*corev1.ServiceAccount)
	nfdRole = decodeManifest(nfdrole).(*rbacv1.ClusterRole)
	nfdRoleBinding = decodeManifest(nfdrolebinding).(*rbacv1.ClusterRoleBinding)
	nfdDaemonSet = decodeManifest(nfddaemonset).(*appsv1.DaemonSet)
}
