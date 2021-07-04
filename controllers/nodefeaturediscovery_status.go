package controllers

import (
	"context"
	"time"
	"errors"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"github.com/openshift/cluster-nfd-operator/pkq/controller/nodefeaturediscovery/components"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//"github.com/openshift/cluster-nfd-operator/pkq/controller/nodefeaturediscovery/components/daemonset"

// nodeType is either 'worker' or 'master'
type nodeType int

const (
	worker nodeType = 0
	master nodeType = 1
	nfdNamespace    = "openshift-nfd"
)

//	appsv1 "k8s.io/api/apps/v1"
//	"errors"
//	"fmt"

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
	conditionNFDWorkerConfigDegraded    = "NFDWorkerConfigResourceDegraded"
	conditionNFDServiceAccountDegraded  = "NFDServiceAccountDegraded"
	conditionNFDServiceDegraded         = "NFDServiceDegraded"
	conditionNFDWorkerDaemonSetDegraded = "NFDWorkerDaemonSetDegraded"
	conditionNFDMasterDaemonSetDegraded = "NFDMasterDaemonSetDegraded"
	conditionNFDRoleDegraded            = "NFDRoleDegraded"
	conditionNFDRoleBindingDegraded     = "NFDRoleBindingDegraded"

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
)

// updateStatus is used to update the status of a resource (e.g., degraded,
// available, etc.)
func (r *NodeFeatureDiscoveryReconciler) updateStatus(nfd *nfdv1.NodeFeatureDiscovery, conditions []conditionsv1.Condition) error {

	r.Log.Info("HERE IN 'updateStatus' func - INIT")

	// The actual 'nfd' object should *not* be modified when trying to
	// check the object's status. This variable is a dummy variable used
	// to set temporary conditions.
	nfdCopy := nfd.DeepCopy()

	r.Log.Info("HERE IN 'updateStatus' func - COPIED NFD")

	//nfdCopy.Status.Conditions = conditions

	// If a set of conditions exists, then it should be added to the
	// 'nfd' Copy.
	if conditions != nil {
		nfdCopy.Status.Conditions = conditions
		r.Log.Info("HERE IN 'updateStatus' func - COPIED CONDITIONS TO NFD")
	} else {
		r.Log.Info("HERE IN 'updateStatus' func - DID NOT COPY CONDITIONS TO NFD")

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
		r.Log.Info("HERE IN 'updateStatus' func - STATUS WAS NOT MODIFIED")
		return nil
	}
	r.Log.Info("HERE IN 'updateStatus' func - RETURNING STATUS UPDATE")

	//klog.Infof("Updating the NFD status")
	//klog.Infof("Conditions: %v", conditions)
	return r.Status().Update(context.TODO(), nfdCopy)
}

// updateDegradedCondition is used to mark a given resource as "degraded" so that
// the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateDegradedCondition(nfd *nfdv1.NodeFeatureDiscovery, condition string, conditionErr error) (ctrl.Result, error) {
	r.Log.Info("Entering degraded condition func")

	// It is already assumed that the resource has been degraded, so the first
	// step is to gather the correct list of conditions.
	var conditionErrMsg string = "Degraded"
	if conditionErr != nil {
		conditionErrMsg = conditionErr.Error()
	}
	conditions := r.getDegradedConditions(condition, conditionErrMsg)
	r.Log.Info("Got degraded conditions")
	if nfd == nil {
		r.Log.Info("nfd is 'nil'")
	}
	if err := r.updateStatus(nfd, conditions); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// updateProgressingCondition is used to mark a given resource as "progressing" so
// that the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateProgressingCondition(nfd *nfdv1.NodeFeatureDiscovery, condition string, conditionErr error) (ctrl.Result, error) {
	r.Log.Info("Entering progressing condition func")

	// It is already assumed that the resource is "progressing," so the first
	// step is to gather the correct list of conditions.
	var conditionErrMsg string = "Progressing"
	if conditionErr != nil {
		conditionErrMsg = conditionErr.Error()
	}
	conditions := r.getProgressingConditions(condition, conditionErrMsg)
	r.Log.Info("Got progressing conditions")
	if nfd == nil {
		r.Log.Info("nfd is 'nil'")
	}
	if err := r.updateStatus(nfd, conditions); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, conditionErr
}

// getAvailableConditions returns a list of conditionsv1.Condition objects and marks
// every condition as FALSE except for conditionsv1.ConditionAvailable so that the
// reconciler can determine that the resource is available.
func (r *NodeFeatureDiscoveryReconciler) getAvailableConditions() []conditionsv1.Condition {
	now := time.Now()
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

//// getConditionOfResource gets the conditions of a specific resource
//func (r *NodeFeatureDiscoveryReconciler) getConditionOfResource(resourceName string, conditions []conditionsv1.Condition) (bool, bool, bool, bool) {
//
//	// Set vars that determine if a status is 'true' or 'false
//	var isAvailable    bool
//	var isUpgradeable  bool
//	var isProgressing  bool
//	var isDegraded     bool
//
//	isAvailable   = false
//	isUpgradeable = false
//	isProgressing = false
//	isDegraded    = false
//
//	// The next step captures the statuses, so use this variable to
//	// keep track of the number of "true" results
//	var numStatusesAsTrue int
//	numStatusesAsTrue = 0
//
//	// Get the resource status via index in the 'Condition' interface
//	for _, c := range conditions {
//
//		r.Log.Info("%q", c)
//
//		// If available, check the status to make sure that it's
//		// set to 'True'
//		if c.Type == "Available" {
//			r.Log.Info("Available")
//			availableStatus := c.Status
//			if availableStatus == "True" {
//				isAvailable = true
//				numStatusesAsTrue++
//			}
//		} else if c.Type == "Upgradeable" {
//			r.Log.Info("Upgradeable")
//			upgradeableStatus := c.Status
//			if upgradeableStatus == "True" {
//				isUpgradeable = true
//				numStatusesAsTrue++
//			}
//		} else if c.Type == "Progressing" {
//			r.Log.Info("Progressing")
//			progressingStatus := c.Status
//			if progressingStatus == "True" {
//				isProgressing = true
//				numStatusesAsTrue++
//			}
//		} else if c.Type == "Degraded" {
//			r.Log.Info("Degraded")
//			degradedStatus := c.Status
//			if degradedStatus == "True" {
//				isDegraded = true
//				numStatusesAsTrue++
//			}
//		}
//	}
//
//	if numStatusesAsTrue == 0 {
//		panic("All statuses are false. There should be at least 1 true type")
//	} else if numStatusesAsTrue > 1 {
//		panic("More than 1 status is set to true")
//	}
//
//	return isAvailable, isUpgradeable, isProgressing, isDegraded
//}

//// resourceIsDegraded determines if a resource is degraded or not
//func (r *NodeFeatureDiscoveryReconciler) resourceIsDegraded(resourceName string, conditions []conditionsv1.Condition) bool {
//	_, _, _, isDegraded := r.getConditionOfResource(resourceName, conditions)
//	if isDegraded == true {
//		return true
//	}
//	return false
//}

//func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBindingConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {
//
//	// Attempt to get the cluster role binding
//	crb, err := components.GetClusterRoleBinding(nfd)
//	if err != nil || crb == nil {
//		messageString := fmt.Sprint(err)
//		return r.getDegradedConditions(conditionReasonNFDDegraded, messageString), errors.New(conditionReasonNFDDegraded)
//	}
//
//	return nil, nil
//}

//func (r *NodeFeatureDiscoveryReconciler) getPodConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {
//
//	// Attempt to get the pod
//	pod, err := components.GetPod(nfd)
//	if err != nil || pod == nil {
//		messageString := fmt.Sprint(err)
//		return r.getDegradedConditions(conditionReasonNFDDegraded, messageString), errors.New(conditionReasonNFDDegraded)
//	}
//
//	return nil, nil
//}

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

// genericStatusGetter is a genric function that interprets the condition
// status of a given resource (e.g., DaemonSet, Service, etc.)
func (r *NodeFeatureDiscoveryReconciler) genericStatusGetter(conditions []*genericResource) resourceStatus {

	// Initialize object which will hold all the results
	rstatus := resourceStatus{isAvailable: false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        false,
		numActiveStatuses: 0,
	}

	var ctype string
	var cstatus string

	// Get the resource status via index in the 'Condition' interface
	for _, c := range conditions {

		ctype = c.Type
		cstatus = c.Status

		log.Info("ctype", "ctype", ctype)
		log.Info("cstatus", "cstatus", cstatus)

		// If available, check the status to make sure that it's
		// set to 'True'
		if ctype == "Available" {
			r.Log.Info("Available")
			if cstatus == "True" {
				rstatus.isAvailable = true
				rstatus.numActiveStatuses++
			}
		} else if ctype == "Upgradeable" {
			r.Log.Info("Upgradeable")
			if cstatus == "True" {
				rstatus.isUpgradeable = true
				rstatus.numActiveStatuses++
			}
		} else if ctype == "Progressing" {
			r.Log.Info("Progressing")
			if cstatus == "True" {
				rstatus.isProgressing = true
				rstatus.numActiveStatuses++
			}
		} else if ctype == "Degraded" {
			r.Log.Info("Degraded")
			if cstatus == "True" {
				rstatus.isDegraded = true
				rstatus.numActiveStatuses++
			}
		} else {
			r.Log.Info("Invalid status")
			panic("Invalid resource status")
		}
	}

	if rstatus.numActiveStatuses == 0 {
		panic("Number of active condition statuses are 0")
	} else if rstatus.numActiveStatuses > 1 {
		panic("Number of active condition statuses are >1. Only one active status is allowed.")
	}

	return rstatus
}

type genericResource struct {
	Type   string
	Status string
}

// getWorkerDaemonSetConditions is a wrapper around
// "getDaemonSetConditions" for ease of calling the
// worker DaemonSet status
func (r *NodeFeatureDiscoveryReconciler) getWorkerDaemonSetConditions(nfd *nfdv1.NodeFeatureDiscovery, ctx context.Context, req ctrl.Request, node nodeType) (resourceStatus, error) {
	return r.__getDaemonSetConditions(nfd, ctx, req, node)
}

// getMasterDaemonSetConditions is a wrapper around
// "getDaemonSetConditions" for ease of calling the
// master DaemonSet status
func (r *NodeFeatureDiscoveryReconciler) getMasterDaemonSetConditions(nfd *nfdv1.NodeFeatureDiscovery, ctx context.Context, req ctrl.Request, node nodeType) (resourceStatus, error) {
	return r.__getDaemonSetConditions(nfd, ctx, req, node)
}

func (r *NodeFeatureDiscoveryReconciler) __getDaemonSetConditions(nfd *nfdv1.NodeFeatureDiscovery, ctx context.Context, req ctrl.Request, node nodeType) (resourceStatus, error) {

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
		err = r.Get(ctx, client.ObjectKey{Namespace: nfdNamespace, Name: "nfd-worker"}, ds)
	} else if node == master {
		err = r.Get(ctx, client.ObjectKey{Namespace: nfdNamespace, Name: "nfd-master"}, ds)
	} else {
		err = errors.New(errorInvalidNodeType)
	}

	if err != nil {
		return rstatus, err
	}

	// Index the DaemonSet status. (Note: there is no "Conditions" array here.)
	dsStatus := ds.Status

	log.Info("ds.Status", "status", dsStatus)

	// Index the relevant values from here
	numberReady            := dsStatus.NumberReady
	currentNumberScheduled := dsStatus.CurrentNumberScheduled
	numberDesired          := dsStatus.DesiredNumberScheduled
	numberUnavailable      := dsStatus.NumberUnavailable

	// If the number desired is zero or the number of unavailable nodes is zero,
	// then we have a problem because we should at least see 1 pod per node
	if numberDesired == 0 {
		if node == worker {
			return rstatus, errors.New(errorNFDWorkerDaemonSetUnknown)
		}
		return rstatus, errors.New(errorNFDMasterDaemonSetUnknown)
	}
	if numberUnavailable > 0 {
		if node == worker {
			return rstatus, errors.New(errorNFDWorkerDaemonSetUnavailableNode)
		}
		return rstatus, errors.New(errorNFDMasterDaemonSetUnavailableNode)
	}

	// If there are none scheduled, then we have a problem because we should
	// at least see 1 pod per node, even after the scheduling happens.
	if currentNumberScheduled == 0 {
		if node == worker {
			return rstatus, errors.New(conditionNFDWorkerDaemonSetDegraded)
		}
		return rstatus, errors.New(conditionNFDMasterDaemonSetDegraded)
	}

	// If we have less than the number of scheduled pods, then the DaemonSet
	// is in progress
	if numberReady < currentNumberScheduled {
		rstatus.isProgressing = true
		rstatus.isDegraded = false
		return rstatus, nil
	}

	// If all nodes are ready, then update the status to be "isAvailable"
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getServiceConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Progressing'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Attempt to get the NFD Operator service
	svc, err := components.GetService(nfd)

	// If there is an error because the 'svc' pointer is nil, then
	// the service is progressing because it isn't ready yet.
	if svc == nil {
		return rstatus, err
	}

	// Get the Service conditions as an array of Service structs
	svcConditions := svc.Status.Conditions

	// Convert results to a list of genericResource objects so that
	// the results can be easily interpreted
	var svccResourcesList []*genericResource
	for _, svcc := range svcConditions {

		var svcItem = new(genericResource)
		svcItem.Type = string(svcc.Type)
		svcItem.Status = string(svcc.Status)

		svccResourcesList = append(svccResourcesList, svcItem)
	}

	// Return
	rstatus = r.genericStatusGetter(svccResourcesList)
	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getWorkerConfigConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Progressing'
	rstatus := resourceStatus{isAvailable: false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Attempt to get the NFD Operator worker config
	wc, err := components.GetWorkerConfig(nfd)

	// If 'wc' is nil, then the resource hasn't been created yet
	if wc == nil {
		return rstatus, err
	}

	// If the NFD operator worker config was found found, then
	// update rstatus so that the worker config resource is
	// marked as 'Available'
	if err == nil {
		rstatus.isDegraded = false
		rstatus.isAvailable = true
	}

	return rstatus, err
}

func (r *NodeFeatureDiscoveryReconciler) getRoleConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Attempt to get the Role
	role, err := components.GetRole(nfd)
	if role == nil {
		return rstatus, err
	}

	// Set the resource to available
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getRoleBindingConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Attempt to get the role binding
	roleb, err := components.GetRoleBinding(nfd)
	if roleb == nil {
		rstatus.isProgressing = true
		rstatus.isDegraded = false
	} else if err != nil {
		return rstatus, err
	}

	// Set the resource to available
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}

func (r *NodeFeatureDiscoveryReconciler) getServiceAccountConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := resourceStatus{
		isAvailable:       false,
		isUpgradeable:     false,
		isProgressing:     false,
		isDegraded:        true,
		numActiveStatuses: 1,
	}

	// Attempt to get the Service Account
	sa, err := components.GetServiceAccount(nfd)
	if sa == nil {
		rstatus.isProgressing = true
		rstatus.isDegraded = false
	} else if err != nil {
		return rstatus, err
	}

	// Set the resource to available
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, nil
}
