package controllers

import (
	"context"
	"time"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"github.com/openshift/cluster-nfd-operator/pkq/controller/nodefeaturediscovery/components"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

	// Unknown error occurred
	conditionFailedGettingKubeletStatus         = "GettingKubeletStatusFailed"
	conditionFailedGettingNFDCustomConfig       = "FailedGettingNFDCustomConfig"
	conditionFailedGettingNFDOperand            = "FailedGettingNFDOperand"
	conditionFailedGettingNFDInstance           = "FailedGettingNFDInstance"
	conditionFailedGettingNFDWorkerConfig       = "FailedGettingNFDWorkerConfig"
	conditionFailedGettingNFDServiceAccount     = "FailedGettingNFDServiceAccount"
	conditionFailedGettingNFDService            = "FailedGettingNFDService"
	conditionFailedGettingNFDDaemonSet          = "FailedGettingNFDDaemonSet"
	conditionFailedGettingNFDClusterRole        = "FailedGettingNFDClusterRole"
	conditionFailedGettingNFDClusterRoleBinding = "FailedGettingNFDClusterRoleBinding"

	// Resource degraded
	conditionNFDWorkerConfigDegraded        = "NFDWorkerConfigResourceDegraded"
	conditionNFDServiceAccountDegraded      = "NFDServiceAccountDegraded"
	conditionNFDServiceDegraded             = "NFDServiceDegraded"
	conditionNFDDaemonSetDegraded           = "NFDDaemonSetDegraded"
	conditionNFDClusterRoleDegraded         = "NFDClusterRoleDegraded"
	conditionNFDClusterRoleBindingDegraded  = "NFDClusterRoleBindingDegraded"
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
	//r.Log.Info("Condition: %s", condition)
	//r.Log.Info("conditionError: %s", conditionErr.Error())
	var conditions []conditionsv1.Condition = r.getDegradedConditions(condition, conditionErr.Error())
	r.Log.Info("Got degraded conditions")
	if nfd == nil {
		r.Log.Info("nfd is 'nil'")
	}
	if err := r.updateStatus(nfd, conditions); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, conditionErr
}

// updateDegradedCondition is used to mark a given resource as "degraded" so that
// the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateProgressingCondition(nfd *nfdv1.NodeFeatureDiscovery, condition string, conditionErr error) (ctrl.Result, error) {
	r.Log.Info("Entering progressing condition func")

	// It is already assumed that the resource has been degraded, so the first
	// step is to gather the correct list of conditions.
	//r.Log.Info("Condition: %s", condition)
	//r.Log.Info("conditionError: %s", conditionErr.Error())
	var conditions []conditionsv1.Condition = r.getProgressingConditions(condition, conditionErr.Error())
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

func (r *NodeFeatureDiscoveryReconciler) getDaemonSetConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Progressing'
	rstatus := resourceStatus{
		isAvailable:   false,
		isUpgradeable: false,
		isProgressing: true,
		isDegraded:    false,
		numActiveStatuses: 1,
	}

	// Attempt to get the daemon set
	ds, err := components.GetDaemonSet(nfd)

	// If there is an error because the 'ds' pointer is nil, then
	// the DaemonSet is progressing because it isn't ready yet.
	if ds == nil {
		return rstatus, err
	}

	// Get the DaemonSet conditions as an array of DaemonSet structs
	dsConditions := ds.Status.Conditions

	// Convert results to a list of genericResource objects so that
	// the results can be easily interpreted
	var dsResourcesList []*genericResource
	for _, dsc := range dsConditions {

		var dsItem = new(genericResource)
		dsItem.Type = string(dsc.Type)
		dsItem.Status = string(dsc.Status)

		dsResourcesList = append(dsResourcesList, dsItem)
	}

	// Return
	rstatus = r.genericStatusGetter(dsResourcesList)
	return rstatus, err
}

func (r *NodeFeatureDiscoveryReconciler) getServiceConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Progressing'
	rstatus := resourceStatus{
		isAvailable:   false,
		isUpgradeable: false,
		isProgressing: true,
		isDegraded:    false,
		numActiveStatuses: 1,
	}

	// Attempt to get the NFD Operator service
	svc, err := components.GetService(nfd)

	// If there is an error because the 'svc' pointer is nil, then
	// the service is progressing because it isn't ready yet.
	if svc == nil {
		return rstatus, err
	}

	// Get the Service conditions as an array of DaemonSet structs
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
	return rstatus, err
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

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Progressing'
	rstatus := resourceStatus{
		isAvailable:   false,
		isUpgradeable: false,
		isProgressing: true,
		isDegraded:    false,
		numActiveStatuses: 1,
	}

	// Attempt to get the Cluster Role
	cr, err := components.GetDaemonSet(nfd)

	// If there is an error because the 'cr' pointer is nil, then
	// the ClusterRole is progressing because it isn't ready yet.
	if cr == nil {
		return rstatus, err
	}

	// Get the ClusterRole conditions as an array of DaemonSet structs
	crConditions := cr.Status.Conditions

	// Convert results to a list of genericResource objects so that
	// the results can be easily interpreted
	var crResourcesList []*genericResource
	for _, crc := range crConditions {

		var crItem = new(genericResource)
		crItem.Type = string(crc.Type)
		crItem.Status = string(crc.Status)

		crResourcesList = append(crResourcesList, crItem)
	}

	// Return
	rstatus = r.genericStatusGetter(crResourcesList)
	return rstatus, err
}

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBindingConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := resourceStatus{
		isAvailable:   false,
		isUpgradeable: false,
		isProgressing: false,
		isDegraded:    true,
		numActiveStatuses: 1,
	}

	// Attempt to get the cluster role binding
	crb, err := components.GetClusterRoleBinding(nfd)
	if crb == nil {
		rstatus.isProgressing = true
		rstatus.isDegraded = false
	} else if err != nil {
		return rstatus, err
	}

	// Set the resource to available
	rstatus.isAvailable = true
	rstatus.isDegraded = false

	return rstatus, err
}

func (r *NodeFeatureDiscoveryReconciler) getServiceAccountConditions(nfd *nfdv1.NodeFeatureDiscovery) (resourceStatus, error) {

	// Initialize Resource Status to 'Degraded'
	rstatus := resourceStatus{
		isAvailable:   false,
		isUpgradeable: false,
		isProgressing: false,
		isDegraded:    true,
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

	return rstatus, err
}
