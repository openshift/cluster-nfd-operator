package nodefeaturediscovery

import (
	"errors"

	secv1 "github.com/openshift/api/security/v1"
	nfdv1alpha1 "github.com/openshift/cluster-nfd-operator/pkg/apis/nfd/v1alpha1"
)

type state interface {
	init(*ReconcileNodeFeatureDiscovery, *nfdv1alpha1.NodeFeatureDiscovery)
	step()
	validate()
	last()
}

type NFD struct {
	resources []Resources
	controls  []controlFunc
	rec       *ReconcileNodeFeatureDiscovery
	ins       *nfdv1alpha1.NodeFeatureDiscovery
	idx       int
}

func addState(n *NFD, path string) error {

	res, ctrl := addResourcesControls(path)

	n.controls = append(n.controls, ctrl)
	n.resources = append(n.resources, res)

	return nil
}

func (n *NFD) init(r *ReconcileNodeFeatureDiscovery,
	i *nfdv1alpha1.NodeFeatureDiscovery) error {
	n.rec = r
	n.ins = i
	n.idx = 0

	secv1.AddToScheme(r.scheme)

	addState(n, "/opt/nfd/master")
	addState(n, "/opt/nfd/worker")

	return nil
}

func (n *NFD) step() error {

	for _, fs := range n.controls[n.idx] {

		stat, err := fs(*n)
		if err != nil {
			return err
		}
		if stat != Ready {
			return errors.New("ResourceNotReady")
		}
	}

	n.idx = n.idx + 1

	return nil
}

func (n NFD) validate() {
	// TODO add custom validation functions
}

func (n NFD) last() bool {
	if n.idx == len(n.controls) {
		return true
	}
	return false
}
