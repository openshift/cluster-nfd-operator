package config

import (
	"os"
	"strconv"

	"github.com/golang/glog"
)

const (
	nodeFeatureDiscoveryImageDefault string = "quay.io/openshift/origin-node-feature-discovery:4.6"
	operatorNameDefault              string = "node-feature-discovery"
	operatorNamespaceDefault         string = "openshift-nfd-operator"
	resyncPeriodDefault              int64  = 600
)

// NodeFeatureDiscoveryImage returns the operator's operand/tuned image path.
func NodeFeatureDiscoveryImage() string {
	nodeFeatureDiscoveryImage := os.Getenv("NODE_FEATURE_DISCOVERY_IMAGE")

	if len(nodeFeatureDiscoveryImage) > 0 {
		return nodeFeatureDiscoveryImage
	}

	return nodeFeatureDiscoveryImageDefault
}

// OperatorName returns the operator name.
func OperatorName() string {
	operatorName := os.Getenv("OPERATOR_NAME")

	if len(operatorName) > 0 {
		return operatorName
	}

	return operatorNameDefault
}

// OperatorNamespace returns the operator namespace.
func OperatorNamespace() string {
	operatorNamespace := os.Getenv("WATCH_NAMESPACE")

	if len(operatorNamespace) > 0 {
		return operatorNamespace
	}

	return operatorNamespaceDefault
}

// ResyncPeriod returns the configured or default Reconcile period.
func ResyncPeriod() int64 {
	resyncPeriodDuration := resyncPeriodDefault
	resyncPeriodEnv := os.Getenv("RESYNC_PERIOD")

	if len(resyncPeriodEnv) > 0 {
		var err error
		resyncPeriodDuration, err = strconv.ParseInt(resyncPeriodEnv, 10, 64)
		if err != nil {
			glog.Errorf("Cannot parse RESYNC_PERIOD (%s), using %d", resyncPeriodEnv, resyncPeriodDefault)
			resyncPeriodDuration = resyncPeriodDefault
		}
	}
	return resyncPeriodDuration
}
