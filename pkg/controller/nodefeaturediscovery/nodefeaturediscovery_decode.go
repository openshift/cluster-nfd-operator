package nodefeaturediscovery

import (
	kappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	appsv1 "github.com/openshift/api/apps/v1"
        authorizationv1 "github.com/openshift/api/authorization/v1"
        buildv1 "github.com/openshift/api/build/v1"
        imagev1 "github.com/openshift/api/image/v1"
        networkv1 "github.com/openshift/api/network/v1"
        oauthv1 "github.com/openshift/api/oauth/v1"
        projectv1 "github.com/openshift/api/project/v1"
        quotav1 "github.com/openshift/api/quota/v1"
        routev1 "github.com/openshift/api/route/v1"
        securityv1 "github.com/openshift/api/security/v1"
        templatev1 "github.com/openshift/api/template/v1"
        userv1 "github.com/openshift/api/user/v1"
	
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	"log"
)

//var nfdNameSpace *corev1.Namespace
var nfdServiceAccount corev1.ServiceAccount
var nfdClusterRole *rbacv1.ClusterRole
var nfdClusterRoleBinding *rbacv1.ClusterRoleBinding
var nfdSecurityContextConstraint *securityv1.SecurityContextConstraints
var nfdDaemonSet *kappsv1.DaemonSet

func decodeManifest(yaml string) interface{} {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(yaml), nil, nil)
	if err != nil {
		log.Printf("Error decoding manifest %v\n", err)
		return nil
	}
	return obj
}

func init() {
	// The Kubernetes Go client (nested within the OpenShift Go client)
        // automatically registers its types in scheme.Scheme, however the
        // additional OpenShift types must be registered manually.  AddToScheme
        // registers the API group types (e.g. route.openshift.io/v1, Route) only.
        appsv1.AddToScheme(scheme.Scheme)
        authorizationv1.AddToScheme(scheme.Scheme)
        buildv1.AddToScheme(scheme.Scheme)
        imagev1.AddToScheme(scheme.Scheme)
        networkv1.AddToScheme(scheme.Scheme)
        oauthv1.AddToScheme(scheme.Scheme)
        projectv1.AddToScheme(scheme.Scheme)
        quotav1.AddToScheme(scheme.Scheme)
        routev1.AddToScheme(scheme.Scheme)
        securityv1.AddToScheme(scheme.Scheme)
        templatev1.AddToScheme(scheme.Scheme)
        userv1.AddToScheme(scheme.Scheme)


	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
                scheme.Scheme)

	_, _, err := s.Decode(nfdserviceaccount, nil, &nfdServiceAccount)
	if err != nil { panic(err) }

	

	

	//	nfdServiceAccount = decodeManifest(nfdserviceaccount).(*corev1.ServiceAccount)
//	nfdClusterRole = decodeManifest(nfdclusterrole).(*rbacv1.ClusterRole)
//	nfdClusterRoleBinding = decodeManifest(nfdclusterrolebinding).(*rbacv1.ClusterRoleBinding)
//	nfdSecurityContextConstraint =  decodeManifest(nfdsecuritycontextconstraint).()
//	nfdDaemonSet = decodeManifest(nfddaemonset).(*appsv1.DaemonSet)
}
