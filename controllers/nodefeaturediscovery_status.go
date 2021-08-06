package controllers

import (
	"context"
	"errors"
	"time"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	nfdMetrics "github.com/openshift/cluster-nfd-operator/pkg/metrics"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	// Condition failed/degraded messages
	conditionReasonValidationFailed         = "ValidationFailed"
	conditionReasonComponentsCreationFailed = "ComponentCreationFailed"
	conditionReasonNFDDegraded              = "NFDDegraded"
	conditionFailedGettingNFDStatus         = "GettingNFDStatusFailed"
	conditionKubeletFailed                  = "KubeletConfig failure"

	// Resource is missing
	conditionFailedGettingKubeletStatus      = "GettingKubeletStatusFailed"
	conditionFailedGettingNFDCustomConfig    = "FailedGettingNFDCustomConfig"
	conditionFailedGettingNFDOperand         = "FailedGettingNFDOperand"
	conditionFailedGettingNFDInstance        = "FailedGettingNFDInstance"
	conditionFailedGettingNFDWorkerConfig    = "FailedGettingNFDWorkerConfig"
	conditionFailedGettingNFDServiceAccount  = "FailedGettingNFDServiceAccount"
	conditionFailedGettingNFDService         = "FailedGettingNFDService"
	conditionFailedGettingNFDWorkerDaemonSet = "FailedGettingNFDWorkerDaemonSet"
	conditionFailedGettingNFDMasterDaemonSet = "FailedGettingNFDMasterDaemonSet"
	conditionFailedGettingNFDRole            = "FailedGettingNFDRole"
	conditionFailedGettingNFDRoleBinding     = "FailedGettingNFDRoleBinding"

	// Resource degraded
	conditionNFDWorkerConfigDegraded       = "NFDWorkerConfigResourceDegraded"
	conditionNFDServiceAccountDegraded     = "NFDServiceAccountDegraded"
	conditionNFDServiceDegraded            = "NFDServiceDegraded"
	conditionNFDWorkerDaemonSetDegraded    = "NFDWorkerDaemonSetDegraded"
	conditionNFDMasterDaemonSetDegraded    = "NFDMasterDaemonSetDegraded"
	conditionNFDRoleDegraded               = "NFDRoleDegraded"
	conditionNFDRoleBindingDegraded        = "NFDRoleBindingDegraded"
	conditionNFDClusterRoleDegraded        = "NFDClusterRoleDegraded"
	conditionNFDClusterRoleBindingDegraded = "NFDClusterRoleBindingDegraded"

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
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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
		return ctrl.Result{}, err
	}
	return ctrl.Result{Requeue: true}, nil
}

// updateAvailableCondition is used to mark a given resource as "progressing" so
// that the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateAvailableCondition(nfd *nfdv1.NodeFeatureDiscovery) (ctrl.Result, error) {

	conditions := r.getAvailableConditions()
	if err := r.updateStatus(nfd, conditions); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, errors.New("CouldNotUpdateAvailableConditions")
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
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Get the existing DaemonSet from the reconciler
	ds := &appsv1.DaemonSet{}
	var err error = nil
	if node == worker {
		err = r.Get(ctx, client.ObjectKey{Namespace: nfdNamespace, Name: workerName}, ds)
	} else if node == master {
		err = r.Get(ctx, client.ObjectKey{Namespace: nfdNamespace, Name: masterName}, ds)
	} else {
		err = errors.New(errorInvalidNodeType)
	}

	if err != nil {
		return rstatus, err
	}

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
	if numberUnavailable > 0 {
		rstatus.isProgressing = true
		rstatus.isDegraded = false
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
		rstatus.isProgressing = true
		rstatus.isDegraded = false
		return rstatus, errors.New(warningNumberOfReadyNodesIsLessThanScheduled)
	}

	// If all nodes are ready, then update the status to be "isAvailable"
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getServiceConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Progressing'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Get the existing Service from the reconciler
	svc := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{Namespace: nfdNamespace, Name: masterName}, svc)

	// If the Service could not be obtained, then it is degraded
	if err != nil {
		return rstatus, errors.New(conditionNFDServiceDegraded)
	}

	// If we could get the Service, then it is not empty and it exists
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil

}

func (r *NodeFeatureDiscoveryReconciler) getWorkerConfigConditions(n NFD) (resourceStatus, error) {

	// Initialize Resource Status to 'Progressing'
	rstatus := resourceStatus{isAvailable: false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}
	// Get the existing ConfigMap from the reconciler
	wc := n.ins.Spec.WorkerConfig.ConfigData

	// If 'wc' is nil, then the resource hasn't been (re)created yet
	if wc == "" {
		return rstatus, errors.New(conditionNFDWorkerConfigDegraded)
	}

	// If we could get the WorkerConfig, then it is not empty and it exists
	rstatus.isDegraded = false
	rstatus.isAvailable = true

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getRoleConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Get the existing Role from the reconciler
	role := &rbacv1.Role{}
	err := r.Get(ctx, client.ObjectKey{Namespace: nfdNamespace, Name: workerName}, role)

	// If 'role' is nil, then it hasn't been (re)created yet
	if err != nil {
		return rstatus, errors.New(conditionNFDRoleDegraded)
	}

	// Set the resource to available
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getRoleBindingConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Get the existing RoleBinding from the reconciler
	rb := &rbacv1.RoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Namespace: nfdNamespace, Name: workerName}, rb)

	// If 'rb' is nil, then it hasn't been (re)created yet
	if err != nil {
		return rstatus, errors.New(conditionNFDRoleBindingDegraded)
	}

	// Set the resource to available
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Get the existing ClusterRole from the reconciler
	clusterRole := &rbacv1.ClusterRole{}
	err := r.Get(ctx, client.ObjectKey{Namespace: "", Name: masterName}, clusterRole)

	// If 'clusterRole' is nil, then it hasn't been (re)created yet
	if err != nil {
		return rstatus, errors.New(conditionNFDClusterRoleDegraded)
	}

	// Set the resource to available
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBindingConditions(ctx context.Context) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Get the existing ClusterRoleBinding from the reconciler
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Namespace: "", Name: masterName}, clusterRoleBinding)

	// If 'clusterRole' is nil, then it hasn't been (re)created yet
	if err != nil {
		return rstatus, errors.New(conditionNFDClusterRoleBindingDegraded)
	}

	// Set the resource to available
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getServiceAccountConditions(ctx context.Context) (resourceStatus, error) {

	// initialize resource status to 'degraded'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Attempt to get the service account from the reconciler
	sa := &corev1.ServiceAccount{}
	err := r.Get(ctx, client.ObjectKey{Namespace: nfdNamespace, Name: masterName}, sa)

	// if 'sa' is nil, then it hasn't been (re)created yet
	if err != nil {
		nfdMetrics.Degraded(true)
		return rstatus, errors.New(conditionNFDServiceAccountDegraded)
	}

	// set the resource to available
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}
