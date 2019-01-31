package nodefeaturediscovery

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	nfdv1alpha1 "github.com/openshift/cluster-nfd-operator/pkg/apis/nfd/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type controlFunc []func(n NFD) error

type state interface {
	init(*ReconcileNodeFeatureDiscovery, *nfdv1alpha1.NodeFeatureDiscovery)
	step()
	validate()
	last()
}

type Resources struct {
	ServiceAccount     corev1.ServiceAccount
	Role               rbacv1.Role
	RoleBinding        rbacv1.RoleBinding
	ClusterRole        rbacv1.ClusterRole
	ClusterRoleBinding rbacv1.ClusterRoleBinding
	ConfigMap          corev1.ConfigMap
	DaemonSet          appsv1.DaemonSet
	Pod                corev1.Pod
	Service            corev1.Service
}

type NFD struct {
	resources []Resources
	controls  []controlFunc
	rec       *ReconcileNodeFeatureDiscovery
	ins       *nfdv1alpha1.NodeFeatureDiscovery
	idx       int
}

func addControls() controlFunc {
	ctrl := controlFunc{}
	ctrl = append(ctrl, ServiceAccount)
	return ctrl
}

//------------------------------------------------------------------------------
type assetsFromFile []byte

var manifests []assetsFromFile

func filePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func getAssetsFrom(path string) []assetsFromFile {

	manifests := []assetsFromFile{}
	assets := path
	files, err := filePathWalkDir(assets)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		buffer, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		manifests = append(manifests, buffer)
	}
	return manifests
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func addResources(path string) Resources {
	res := Resources{}

	manifests := getAssetsFrom(path)

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
		scheme.Scheme)
	reg, _ := regexp.Compile(`\b(\w*kind:\w*)\B.*\b`)

	for _, m := range manifests {
		kind := reg.FindString(string(m))
		slce := strings.Split(kind, ":")
		kind = strings.TrimSpace(slce[1])

		switch kind {
		case "ServiceAccount":
			_, _, err := s.Decode(m, nil, &res.ServiceAccount)
			panicIfError(err)
		case "ClusterRole":
			_, _, err := s.Decode(m, nil, &res.ClusterRole)
			panicIfError(err)
		case "ClusterRoleBinding":
			_, _, err := s.Decode(m, nil, &res.ClusterRoleBinding)
			panicIfError(err)
		case "ConfigMap":
			_, _, err := s.Decode(m, nil, &res.ConfigMap)
			panicIfError(err)
		case "DaemonSet":
			_, _, err := s.Decode(m, nil, &res.DaemonSet)
			panicIfError(err)
		default:
			log.Info("Resource unknown will not decode: ", m)
		}

	}

	return res
}

//------------------------------------------------------------------------------

func addState(n *NFD, path string) error {

	n.controls = append(n.controls, addControls())
	n.resources = append(n.resources, addResources(path))

	return nil
}

func (n *NFD) init(r *ReconcileNodeFeatureDiscovery,
	i *nfdv1alpha1.NodeFeatureDiscovery) error {
	n.rec = r
	n.ins = i
	n.idx = 0

	err := addState(n, "/opt/nfd/master")
	if err != nil {
		return err
	}

	return nil
}

func (n *NFD) step() error {

	for _, fs := range n.controls[n.idx] {
		err := fs(*n)
		if err != nil {
			return err
		}
	}

	n.idx = n.idx + 1

	return nil
}

func (n NFD) validate() {

}

func (n NFD) last() bool {
	if n.idx == len(n.controls) {
		return true
	}
	return false
}
