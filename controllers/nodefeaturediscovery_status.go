package controllers

import (
	"context"
	"errors"
	"time"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	nfdMetrics "github.com/openshift/cluster-nfd-operator/pkg/metrics"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// nodeType is either 'worker' or 'master'
type nodeType int

const (
	worker       nodeType = 0
	master       nodeType = 1
	nfdNamespace          = "openshift-nfd"
	workerName            = "nfd-worker"
	masterName            = "nfd-master"
)

const (
	// Resource is missing
	conditionFailedGettingKubeletStatus           = "GettingKubeletStatusFailed"
	conditionFailedGettingNFDCustomConfig         = "FailedGettingNFDCustomConfig"
	conditionFailedGettingNFDOperand              = "FailedGettingNFDOperand"
	conditionFailedGettingNFDInstance             = "FailedGettingNFDInstance"
	conditionFailedGettingNFDWorkerConfig         = "FailedGettingNFDWorkerConfig"
	conditionFailedGettingNFDWorkerServiceAccount = "FailedGettingNFDServiceAccount"
	conditionFailedGettingNFDMasterServiceAccount = "FailedGettingNFDServiceAccount"
	conditionFailedGettingNFDService              = "FailedGettingNFDService"
	conditionFailedGettingNFDWorkerDaemonSet      = "FailedGettingNFDWorkerDaemonSet"
	conditionFailedGettingNFDMasterDaemonSet      = "FailedGettingNFDMasterDaemonSet"
	conditionFailedGettingNFDRole                 = "FailedGettingNFDRole"
	conditionFailedGettingNFDRoleBinding          = "FailedGettingNFDRoleBinding"

	// Resource degraded
	conditionNFDWorkerConfigDegraded         = "NFDWorkerConfigResourceDegraded"
	conditionNFDWorkerServiceAccountDegraded = "NFDWorkerServiceAccountDegraded"
	conditionNFDMasterServiceAccountDegraded = "NFDMasterServiceAccountDegraded"
	conditionNFDServiceDegraded              = "NFDServiceDegraded"
	conditionNFDWorkerDaemonSetDegraded      = "NFDWorkerDaemonSetDegraded"
	conditionNFDMasterDaemonSetDegraded      = "NFDMasterDaemonSetDegraded"
	conditionNFDRoleDegraded                 = "NFDRoleDegraded"
	conditionNFDRoleBindingDegraded          = "NFDRoleBindingDegraded"
	conditionNFDClusterRoleDegraded          = "NFDClusterRoleDegraded"
	conditionNFDClusterRoleBindingDegraded   = "NFDClusterRoleBindingDegraded"

	// Unknown errors. (These occur when the error is unknown.)
	errorNFDWorkerDaemonSetUnknown = "NFDWorkerDaemonSetCorrupted"
	errorNFDMasterDaemonSetUnknown = "NFDMasterDaemonSetCorrupted"

	// Unavailable node errors. (These are triggered when one or
	// more nodes are unavailable.)
	errorNFDWorkerDaemonSetUnavailableNode = "NFDWorkerDaemonSetUnavailableNode"
	errorNFDMasterDaemonSetUnavailableNode = "NFDMasterDaemonSetUnavailableNode"

	// Invalid node type. (Denotes that the node should be either
	// 'worker' or 'master')
	errorInvalidNodeType = "InvalidNodeTypeSelected"

	// More nodes are listed as "ready" than selected
	errorTooManyNFDWorkerDaemonSetReadyNodes = "NFDWorkerDaemonSetHasMoreNodesThanScheduled"
	errorTooManyNFDMasterDaemonSetReadyNodes = "NFDMasterDaemonSetHasMoreNodesThanScheduled"

	// DaemonSet warnings (for "Progressing" conditions)
	warningNumberOfReadyNodesIsLessThanScheduled = "warningNumberOfReadyNodesIsLessThanScheduled"
	warningNFDWorkerDaemonSetProgressing         = "warningNFDWorkerDaemonSetProgressing"
	warningNFDMasterDaemonSetProgressing         = "warningNFDMasterDaemonSetProgressing"
)

// updateStatus is used to update the status of a resource (e.g., degraded,
// available, etc.)
func (r *NodeFeatureDiscoveryReconciler) updateStatus(nfd *nfdv1.NodeFeatureDiscovery, conditions []conditionsv1.Condition) error {

	// The actual 'nfd' object should *not* be modified when trying to
	// check the object's status. This variable is a dummy variable used
	// to set temporary conditions.
	nfdCopy := nfd.DeepCopy()

	// If a set of conditions exists, then it should be added to the
	// 'nfd' Copy.
	if conditions != nil {
		nfdCopy.Status.Conditions = conditions
	}

	// Next step is to check if we need to update the status
	modified := false

	// Because there are only four possible conditions (degraded, available,
	// updatable, and progressing), it isn't necessary to check if old
	// conditions should be removed.
	for _, newCondition := range nfdCopy.Status.Conditions {
		oldCondition := conditionsv1.FindStatusCondition(nfd.Status.Conditions, newCondition.Type)
		if oldCondition == nil {
			modified = true
			break
		}
		// Ignore timestamps to avoid infinite reconcile loops
		if oldCondition.Status != newCondition.Status ||
			oldCondition.Reason != newCondition.Reason ||
			oldCondition.Message != newCondition.Message {
			modified = true
			break
		}
	}

	// If nothing has been modified, then return nothing. Even if the list
	// of 'conditions' is not empty, it should not be counted as an update
	// if it was already counted as an update before.
	if !modified {
		return nil
	}
	return r.Status().Update(context.TODO(), nfdCopy)
}

// updateDegradedCondition is used to mark a given resource as "degraded" so that
// the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateDegradedCondition(nfd *nfdv1.NodeFeatureDiscovery, condition string, conditionErr error) (ctrl.Result, error) {

	nfdMetrics.Degraded(true)
	// It is already assumed that the resource has been degraded, so the first
	// step is to gather the correct list of conditions.
	var conditionErrMsg string = "Degraded"
	if conditionErr != nil {
		conditionErrMsg = conditionErr.Error()
	}
	conditions := r.getDegradedConditions(condition, conditionErrMsg)
	if err := r.updateStatus(nfd, conditions); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

// updateProgressingCondition is used to mark a given resource as "progressing" so
// that the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateProgressingCondition(nfd *nfdv1.NodeFeatureDiscovery, condition string, conditionErr error) (ctrl.Result, error) {

	// It is already assumed that the resource is "progressing," so the first
	// step is to gather the correct list of conditions.
	var conditionErrMsg string = "Progressing"
	if conditionErr != nil {
		conditionErrMsg = conditionErr.Error()
	}
	conditions := r.getProgressingConditions(condition, conditionErrMsg)
	if err := r.updateStatus(nfd, conditions); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, nil
}

// updateAvailableCondition is used to mark a given resource as "progressing" so
// that the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateAvailableCondition(nfd *nfdv1.NodeFeatureDiscovery) (ctrl.Result, error) {

	conditions := r.getAvailableConditions()
	if err := r.updateStatus(nfd, conditions); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, errors.New("CouldNotUpdateAvailableConditions")
}

// getAvailableConditions returns a list of conditionsv1.Condition objects and marks
// every condition as FALSE except for conditionsv1.ConditionAvailable so that the
// reconciler can determine that the resource is available.
func (r *NodeFeatureDiscoveryReconciler) getAvailableConditions() []conditionsv1.Condition {
	now := time.Now()
	nfdMetrics.Degraded(false)
	return []conditionsv1.Condition{
		{
			Type:               conditionsv1.ConditionAvailable,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
	}
}

// getDegradedConditions returns a list of conditionsv1.Condition objects and marks
// every condition as FALSE except for conditionsv1.ConditionDegraded so that the
// reconciler can determine that the resource is degraded.
func (r *NodeFeatureDiscoveryReconciler) getDegradedConditions(reason string, message string) []conditionsv1.Condition {
	now := time.Now()
	return []conditionsv1.Condition{
		{
			Type:               conditionsv1.ConditionAvailable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			LastHeartbeatTime:  metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
	}
}

// getProgressingConditions returns a list of conditionsv1.Condition objects and marks
// every condition as FALSE except for conditionsv1.ConditionProgressing so that the
// reconciler can determine that the resource is progressing.
func (r *NodeFeatureDiscoveryReconciler) getProgressingConditions(reason string, message string) []conditionsv1.Condition {
	now := time.Now()
	return []conditionsv1.Condition{
		{
			Type:               conditionsv1.ConditionAvailable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionUpgradeable,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
		{
			Type:               conditionsv1.ConditionProgressing,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Time{Time: now},
			Reason:             reason,
			Message:            message,
		},
		{
			Type:               conditionsv1.ConditionDegraded,
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Time{Time: now},
		},
	}
}

// The status of the resource (available, upgradeable, progressing, or
// degraded).
type resourceStatus struct {

	// Is the resource available, upgradable, etc.?
	isAvailable   bool
	isUpgradeable bool
	isProgressing bool
	isDegraded    bool

	// How many statuses are set to 'true'?
	numActiveStatuses int
}

// initializeDegradedStatus initializes the status struct to degraded
func initializeDegradedStatus() resourceStatus {
	return resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}
}

// setStatusAsAvailable sets the current resource status
// as "isAvailable". This function is called after all
// resource status conditions have been checked.
func setStatusAsAvailable(rstatus *resourceStatus) {
	rstatus.isAvailable = true
	rstatus.isUpgradeable = false
	rstatus.isProgressing = false
	rstatus.isDegraded = false
	rstatus.numActiveStatuses = 1
}

// setStatusAsProgressing sets the current resource status
// as "isProgressing". This function is called after all
// resource status conditions have been checked.
func setStatusAsProgressing(rstatus *resourceStatus) {
	rstatus.isAvailable = false
	rstatus.isUpgradeable = false
	rstatus.isProgressing = true
	rstatus.isDegraded = false
	rstatus.numActiveStatuses = 1
}

// getWorkerDaemonSetConditions is a wrapper around
// "getDaemonSetConditions" for ease of calling the
// worker DaemonSet status
func (r *NodeFeatureDiscoveryReconciler) getWorkerDaemonSetConditions(ctx context.Context) (resourceStatus, error) {
	return r.getDaemonSetConditions(ctx, worker)
}

// getMasterDaemonSetConditions is a wrapper around
// "getDaemonSetConditions" for ease of calling the
// master DaemonSet status
func (r *NodeFeatureDiscoveryReconciler) getMasterDaemonSetConditions(ctx context.Context) (resourceStatus, error) {
	return r.getDaemonSetConditions(ctx, master)
}

func (r *NodeFeatureDiscoveryReconciler) getDaemonSetConditions(ctx context.Context, node nodeType) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := initializeDegradedStatus()

	// Get the existing DaemonSet from the reconciler
	var err error = nil
	var dsName string
	if node == worker {
		dsName = workerName
	} else if node == master {
		dsName = masterName
	} else {
		err = errors.New(errorInvalidNodeType)
	}

	if err != nil {
		return rstatus, err
	}

	// Get the current DaemonSet from the reconciler
	ds, err := r.getDaemonSet(ctx, nfdNamespace, dsName)

	// Index the DaemonSet status. (Note: there is no "Conditions" array here.)
	dsStatus := ds.Status

	// Index the relevant values from here
	numberReady := dsStatus.NumberReady
	currentNumberScheduled := dsStatus.CurrentNumberScheduled
	numberDesired := dsStatus.DesiredNumberScheduled
	numberUnavailable := dsStatus.NumberUnavailable

	// If the number desired is zero or the number of unavailable nodes is zero,
	// then we have a problem because we should at least see 1 pod per node
	if numberDesired == 0 {
		if node == worker {
			return rstatus, errors.New(errorNFDWorkerDaemonSetUnknown)
		}
		return rstatus, errors.New(errorNFDMasterDaemonSetUnknown)
	}

	// If one or more pods is listed as "Unavailable", then it means the
	// DaemonSet is currently progressing, and neither "Available" or "Degraded"
	if numberUnavailable > 0 {
		setStatusAsProgressing(&rstatus)
		if node == worker {
			return rstatus, errors.New(warningNFDWorkerDaemonSetProgressing)
		}
		return rstatus, errors.New(warningNFDMasterDaemonSetProgressing)
	}

	// If there are none scheduled, then we have a problem because we should
	// at least see 1 pod per node, even after the scheduling happens.
	if currentNumberScheduled == 0 {
		if node == worker {
			return rstatus, errors.New(conditionNFDWorkerDaemonSetDegraded)
		}
		return rstatus, errors.New(conditionNFDMasterDaemonSetDegraded)
	}

	// Just check in case the number of "ready" nodes is greater than the
	// number of scheduled ones (for whatever reason)
	if numberReady > currentNumberScheduled {
		rstatus.isDegraded = false
		rstatus.numActiveStatuses = 0
		if node == worker {
			return rstatus, errors.New(errorTooManyNFDWorkerDaemonSetReadyNodes)
		}
		return rstatus, errors.New(errorTooManyNFDMasterDaemonSetReadyNodes)
	}

	// If we have less than the number of scheduled pods, then the DaemonSet
	// is in progress
	if numberReady < currentNumberScheduled {
		setStatusAsProgressing(&rstatus)
		return rstatus, errors.New(warningNumberOfReadyNodesIsLessThanScheduled)
	}

	// If all nodes are ready, then update the status to be "isAvailable"
	setStatusAsAvailable(&rstatus)

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getServiceConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := initializeDegradedStatus()

	// Get the existing Service from the reconciler
	_, err := r.getService(ctx, nfdNamespace, masterName)

	// If the Service could not be obtained, then it is degraded
	if err != nil {
		return rstatus, errors.New(conditionNFDServiceDegraded)
	}

	// If we could get the Service, then it is not empty and it exists
	setStatusAsAvailable(&rstatus)

	return rstatus, nil

}

func (r *NodeFeatureDiscoveryReconciler) getWorkerConfigConditions(n NFD) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := initializeDegradedStatus()

	// Get the existing ConfigMap from the reconciler
	wc := n.ins.Spec.WorkerConfig.ConfigData

	// If 'wc' is nil, then the resource hasn't been (re)created yet
	if wc == "" {
		return rstatus, errors.New(conditionNFDWorkerConfigDegraded)
	}

	// If we could get the WorkerConfig, then it is not empty and it exists
	setStatusAsAvailable(&rstatus)

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getRoleConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := initializeDegradedStatus()

	// Get the existing Role from the reconciler
	_, err := r.getRole(ctx, nfdNamespace, workerName)

	// If 'role' is nil, then it hasn't been (re)created yet
	if err != nil {
		return rstatus, errors.New(conditionNFDRoleDegraded)
	}

	// Set the resource to available
	setStatusAsAvailable(&rstatus)

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getRoleBindingConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := initializeDegradedStatus()

	// Get the existing RoleBinding from the reconciler
	_, err := r.getRoleBinding(ctx, nfdNamespace, workerName)

	// If the error is not nil, then it hasn't been (re)created yet
	if err != nil {
		return rstatus, errors.New(conditionNFDRoleBindingDegraded)
	}

	// Set the resource to available
	setStatusAsAvailable(&rstatus)

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := initializeDegradedStatus()

	// Get the existing ClusterRole from the reconciler
	_, err := r.getClusterRole(ctx, "", masterName)

	// If 'clusterRole' is nil, then it hasn't been (re)created yet
	if err != nil {
		return rstatus, errors.New(conditionNFDClusterRoleDegraded)
	}

	// Set the resource to available
	setStatusAsAvailable(&rstatus)

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBindingConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := initializeDegradedStatus()

	// Get the existing ClusterRoleBinding from the reconciler
	_, err := r.getClusterRoleBinding(ctx, "", masterName)

	// If 'clusterRole' is nil, then it hasn't been (re)created yet
	if err != nil {
		return rstatus, errors.New(conditionNFDClusterRoleBindingDegraded)
	}

	// Set the resource to available
	setStatusAsAvailable(&rstatus)

	return rstatus, nil
}

// getWorkerDaemonSetServiceAccount is a wrapper around
// "getServiceAccountConditions" for ease of calling the
// worker ServiceAccount status
func (r *NodeFeatureDiscoveryReconciler) getWorkerServiceAccountConditions(ctx context.Context) (resourceStatus, error) {
	return r.getServiceAccountConditions(ctx, worker)
}

// getMasterDaemonSetServiceAccount is a wrapper around
// "getServiceAccountConditions" for ease of calling the
// master ServiceAccount status
func (r *NodeFeatureDiscoveryReconciler) getMasterServiceAccountConditions(ctx context.Context) (resourceStatus, error) {
	return r.getServiceAccountConditions(ctx, worker)
}

func (r *NodeFeatureDiscoveryReconciler) getServiceAccountConditions(ctx context.Context, node nodeType) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := initializeDegradedStatus()

	var err error = nil
	var saName string
	if node == worker {
		saName = workerName
	} else if node == master {
		saName = masterName
	} else {
		err = errors.New(errorInvalidNodeType)
	}

	if err != nil {
		return rstatus, err
	}

	// Attempt to get the service account from the reconciler
	_, err = r.getServiceAccount(ctx, nfdNamespace, saName)

	// if 'sa' is nil, then it hasn't been (re)created yet
	if err != nil {
		nfdMetrics.Degraded(true)
		if node == worker {
			return rstatus, errors.New(conditionNFDWorkerServiceAccountDegraded)
		}
		return rstatus, errors.New(conditionNFDMasterServiceAccountDegraded)
	}

	// Set the Resource Status to Available
	setStatusAsAvailable(&rstatus)

	return rstatus, nil
}
