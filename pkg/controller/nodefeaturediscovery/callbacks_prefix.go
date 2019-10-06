package nodefeaturediscovery

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func prefixNFDmaster(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery) error {
	return setDSimage(obj, r, os.Getenv("NODE_FEATURE_DISCOVERY_IMAGE"), "nfd-master")
}

func prefixNFDworker(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery) error {
	return setDSimage(obj, r, os.Getenv("NODE_FEATURE_DISCOVERY_IMAGE"), "nfd-worker")
}

func setDSimage(obj *unstructured.Unstructured, r *ReconcileNodeFeatureDiscovery, value string, name string) error {

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "template", "spec", "containers")
	checkNestedFields(found, err)

	for _, container := range containers {
		switch container := container.(type) {
		case map[string]interface{}:
			if container["name"] == name {
				img, found, err := unstructured.NestedString(container, "image")
				checkNestedFields(found, err)
				img = value
				err = unstructured.SetNestedField(container, img, "image")
				checkNestedFields(true, err)
			}
		default:
			panic(fmt.Errorf("cannot extract name,image from %T", container))
		}
	}

	err = unstructured.SetNestedSlice(obj.Object, containers,
		"spec", "template", "spec", "containers")
	checkNestedFields(true, err)

	return nil
}
