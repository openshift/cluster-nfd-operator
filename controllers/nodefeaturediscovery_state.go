/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"errors"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
)

// NFD holds the needed information to watch from the Controller. The
// following descriptions elaborate on each field in this struct:
type NFD struct {
	// resources contains information about NFD's resources.
	resources []Resources

	// controls is a list that contains the status of an NFD resource
	// as being Ready (=0) or NotReady (=1)
	controls []controlFunc

	// rec represents the NFD reconciler struct used for reconciliation
	rec *NodeFeatureDiscoveryReconciler

	// ins is the NodeFeatureDiscovery struct that contains the Schema
	// for the nodefeaturediscoveries API
	ins *nfdv1.NodeFeatureDiscovery

	// idx is the index that is used to step through the 'controls' list
	// and is set to 0 upon calling 'init()'
	idx int
}

// addState takes a given path and finds resources in that path,
// then appends a list of ctrl's functions to the NFD object's
// 'controls' field and adds the list of resources found to
// 'n.resources'
func (n *NFD) addState(path string) {
	res, ctrl := addResourcesControls(path)
	n.controls = append(n.controls, ctrl)
	n.resources = append(n.resources, res)
}

// init initializes an NFD object by populating the fields before
// attempting to run any kind of check.
func (n *NFD) init(
	r *NodeFeatureDiscoveryReconciler,
	i *nfdv1.NodeFeatureDiscovery,
) {
	n.rec = r
	n.ins = i
	n.idx = 0
	if len(n.controls) == 0 {
		n.addState("/opt/nfd/master")
		n.addState("/opt/nfd/worker")
		n.addState("/opt/nfd/topologyupdater")
	}
}

// step steps through the list of functions stored in 'n.controls',
// then attempts to determine if the given resource is Ready or
// NotReady. (See the following file for a list of functions that
// 'n.controls' can take on: ./nodefeaturediscovery_resources.go.)
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

// last checks if the last index equals the number of functions
// stored in n.controls.
func (n *NFD) last() bool {
	return n.idx == len(n.controls)
}
