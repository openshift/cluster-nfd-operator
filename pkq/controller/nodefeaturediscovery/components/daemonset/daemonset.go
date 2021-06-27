package daemonset

import (
	"errors"
	"encoding/json"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
)

// DaemonSet fixed values
const (
	nfdWorker = "nfd-worker"
	nfdMaster = "nfd-master"
	numExpectedDs  = 2
	dsAPIVersion string = "apps/v1"
)

// Other fixed values
const (
	namespace = "openshift-nfd"
)

// Error messages
const (
	errorUnexpectedDaemonSetName        = "unexpectedDaemonSetName"
	errorFoundDuplicateWorkerDaemonSets = "foundDuplicateWorkerDaemonSets"
	errorFoundDuplicateMasterDaemonSets = "foundDuplicateMasterDaemonSets"
	errorUnexpectedNumDaemonSets        = "unexpectedNumberOfDaemonSetsFound"
	errorFindingSpec                    = "couldNotFindDaemonSetSpec"
	errorFindingStatus                  = "couldNotFindDaemonSetStatus"
)

func Set(nodeType string, nfd *nfdv1.NodeFeatureDiscovery, n NFD) (error) {

	// Attempt to get all the daemon sets
	allDaemonSets, err := getDaemonSets(n)
	if err != nil {
		return err
	}

	// Figure out which one is the worker and which
	// one is the master
	var dsName string
	var rawWorkerDs string
	var rawMasterDs string
	var numWorkers int = 0
	for _, ds := range allDaemonSets {

		// Extract the object meta name so that
		// it is possible to figure out if the
		// current DaemonSet object is a worker,
		// master, or none.
		dsName = ds.ObjectMeta.Name

		if dsName == nfdWorker {
			rawWorkerDs = ds
			numWorkers++

		} else if dsName == nfdMaster {
			rawMasterDs = ds
		} else {
			return errors.New(errorUnexpectedDaemonSetName)
		}
	}

	// Make sure we don't have two workers or two 
	// masters
	if numWorkers == 2 {
		return errors.New(errorFoundDuplicateWorkerDaemonSets)
	}
	if numWorkers == 0 {
		return errors.New(errorFoundDuplicateMasterDaemonSets)
	}

	// Convert Worker json to map
	workerDsRawMap, err := convertRawToMap(rawWorkerDs)
	if err != nil {
		return err
	}

	// Convert Master json to map
	masterDsRawMap, err := convertRawToMap(rawMasterDs)
	if err != nil {
		return err
	}

	// Get the specs
	workerSpec, err := getSpec(workerDsRawMap)
	if err != nil {
		return err
	}
	masterSpec, err := getSpec(masterDsRawMap)
	if err != nil {
		return err
	}

	// Get the status
	workerStatus, err := getStatus(workerDsRawMap)
	if err != nil {
		return err
	}
	masterStatus, err := getStatus(masterDsRawMap)
	if err != nil {
		return err
	}

	// Now set the daemon sets
	wds := &appsv1.DaemonSet {
		TypeMeta: metav1.TypeMeta{
			APIVersion: dsAPIVersion,
			Kind: "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nfdWorker,
			Labels: map[string]string{"app": nfdWorker},
		},
		Spec: workerSpec,
		Status: workerStatus,
	}

	mds := &appsv1.DaemonSet {
		TypeMeta: metav1.TypeMeta{
			APIVersion: dsAPIVersion,
			Kind: "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nfdMaster,
			Labels: map[string]string{"app": nfdMaster},
		},
		Spec: masterSpec,
		Status: masterStatus,
	}

	nfd.Spec.WorkerDaemonSet = wds
	nfd.Spec.MasterDaemonSet = mds

	return nil
}

// getDaemonSets grabs both DaemonSet objects
func getDaemonSets(n NFD) (Resources, error) {

	// Grab all the DaemonSets
	allDaemonSets := n.resources[state].DaemonSet

	// Make sure we only have two daemonsets -- one for
	// the worker and one for the master
	if len(allDaemonSets) != numExpectedDs {
		return allDaemonSets, errors.New(errorUnexpectedNumDaemonSets)
	}
	return allDaemonSets, nil
}

// convertRawToMap converts raw JSON data to a JSON
// 'RawMessage' object
func convertRawToMap(rawManifest string) (map[string]json.RawMessage, error) {

	// Convert json to map
	var dsMap map[string]json.RawMessage
	err = json.Unmarshal([]byte(rawManifest), &dsMap)

	// If there's a problem where the JSON string
	// is empty, stop
	if err != nil {
		return nil, err
	}

	return dsMap, nil
}

// getSpec gets the DaemonSet's spec (for a worker or master)
func getSpec(jsonRawMessageManifest map[string]json.RawMessage) (appsv1.DaemonSetSpec, error) {

	// Initialize selector vars
	var selector            *metav1.LabelSelector
	var matchLabels          map[string]string
	var matchExpressions   []metav1.LabelSelectorRequirement

	// Initialize template vars
	var template             corev1.PodTemplateSpec
	var templatePodSpec      corev1.PodSpec
	var templateObjMeta      metav1.ObjectMeta

	// Initialize update strategy vars
	var updateStrategy       appsv1.DaemonSetUpdateStrategy
	var updateStrategyType   appsv1.DaemonSetUpdateStrategyType
	var rollingUpdate       *appsv1.RollingUpdateDaemonSet

	// Remaining vars
	var minReadySeconds       int32
	var revisionHistoryLimit *int32

	// Attempt to grab the spec or throw an error
	if spec, ok := jsonRawMessageManifest["Spec"]; ok {

		// Get the basic vars first whose types are
		// basic Golang types (i.e., int32 and *int32)
		minReadySeconds = int32(spec["MinReadySeconds"])
		revisionHistoryLimit = &int32(spec["RevisionHistoryLimit"])

		// Now let's grab the selector vars. We don't
		// care about the "matchExpressions" though.
		matchLabels := spec["Selector"]["MatchLabels"]
		selector.MatchLabels      = matchLabels
		selector.MatchExpressions = matchExpressions //empty for now

		// For the template, it's a little more complicated.
		// It is necessary to get the object metadata and the
		// pod spec. However, we don't need all of this info.
		templateObjMeta.APIVersion = dsAPIVersion
		templateObjMeta.Kind       = "DaemonSetSpec"

		specTemplate := spec["Template"]
		templatePodSpec.Volumes            = specTemplate["Volumes"]
		templatePodSpec.ServiceAccountName = specTemplate["ServiceAccountName"]
		templatePodSpec.RestartPolicy      = corev1.RestartPolicy(specTemplate["RestartPolicy"])

		template.OjectMeta = templateObjMeta
		template.PodSpec   = templatePodSpec

		// Get the update strategy too
		specStrategy := spec["Strategy"]
		updateStrategyType = appsv1.DaemonSetUpdateStrategyType(specStrategy["Type"])
		rollingUpdate.MaxUnavailable = *intstr.IntOrString(specStrategy["RollingUpdate"]["MaxUnavailable"])
		rollingUpdate.MaxSurge = *intstr.IntOrString(specStrategy["RollingUpdate"]["MaxSurge"])

		updateStrategy.Type = updateStrategyType
		updateStrategy.RollingUpdate = rollingUpdate

		// Create spec object
		dsSpec := &appsv1.DaemonSetSpec{
			Selector: selector,
			Template: template,
			UpdateStrategy: updateStrategy,
			MinReadySeconds: minReadySeconds,
			RevisionHistoryLimit: revisoinHistoryLimit,
		}

		return dsSpec, nil
	}
	return nil, errors.New(errorFindingDaemonSetSpec)
}

// getStatus gets the status of the daemonset
func getStatus(jsonRawMessageManifest map[string]json.RawMessage) (appsv1.DaemonSetStatus, error) {

	// Attempt to grab the status or throw an error
	if status, ok := jsonRawMessageManifest["Status"]; ok {

		// Everything here is an integer or pointer
		// to an integer, *except* for Conditions
		currentNumberScheduled := status["CurrentNumberScheduled"]
		numberMisscheduled     := status["NumberMisscheduled"]
		desiredNumberScheduled := status["desiredNumberScheduled"]
		numberReady            := status["NumberReady"]
		observedGeneration     := status["ObservedGeneration"]
		updatedNumberScheduled := status["UpdatedNumberScheduled"]
		numberAvailable        := status["NumberAvailable"]
		numberUnavailable      := status["NumberUnavailable"]
		collisionCount         := status["CollisionCount"]

		// Now parse the conditions. For now, ignore the
		// 'LastTransitionTime' var
		conditions := status["Conditions"]
		var daemonSetConditions appsv1.DaemonSetCondition
		daemonSetConditions.Type    = appsv1.DaemonSetConditionType(conditions["Type"])
		daemonSetConditions.Status  = corev1.ConditionStatus(conditions["Status"])
		daemonSetConditions.Reason  = string(conditions["Reason"])
		daemonSetConditions.Message = string(conditions["Message"])

		dsStatus := &appsv1.DaemonSetStatus{
			CurrentNumberScheduled: currentNumberScheduled,
			NumberMisscheduled:     numberMisscheduled,
			DesiredNumberScheduled: desiredNumberScheduled,
			NumberReady:            numberReady,
			ObservedGeneration:     observedGeneration,
			UpdatedNumberScheduled: updatedNumberScheduled,
			NumberAvailable:        numberAvailable,
			NumberUnavailable:      numberUnavailable,
			CollisionCount:         collisionCount,
			Conditions:             daemonSetConditions,
		}

		return dsStatus, nil
	}
	return nil, errors.New(errorFindingDaemonSetStatus)
}
