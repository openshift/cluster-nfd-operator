package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"github.com/openshift/cluster-nfd-operator/pkq/controller/nodefeaturediscovery/components"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// Condition failed/degraded messages
	conditionReasonValidationFailed         = "ValidationFailed"
	conditionReasonComponentsCreationFailed = "ComponentCreationFailed"
	conditionReasonNFDDegraded              = "NFDDegraded"
	conditionFailedGettingNFDStatus         = "GettingNFDStatusFailed"
	conditionKubeletFailed                  = "KubeletConfig failure"
	conditionFailedGettingKubeletStatus     = "GettingKubeletStatusFailed"
	conditionFailedGettingNFDCustomConfig   = "FailedGettingNFDCustomConfig"
	conditionFailedGettingNFDOperand        = "FailedGettingNFDOperand"
	conditionFailedGettingNFDInstance       = "FailedGettingNFDInstance"
	conditionFailedGettingNFDWorkerConfig   = "FailedGettingNFDWorkerConfig"
	conditionFailedGettingNFDServiceAccount = "FailedGettingNFDServiceAccount"
	conditionFailedGettingNFDService        = "FailedGettingNFDService"
	conditionFailedGettingNFDDaemonSet      = "FailedGettingNFDDaemonSet"
	conditionFailedGettingNFDClusterRole    = "FailedGettingNFDClusterRole"

	// Condition references
	conditionAvailable   = conditionsv1.ConditionAvailable
	conditionUpgradeable = conditionsv1.ConditionUpgradeable
	conditionDegraded    = conditionsv1.ConditionDegraded
	conditionProgressing = conditionsv1.ConditionProgressing
)

// updateStatus is used to update the status of a resource (e.g., degraded,
// available, etc.)
func (r *NodeFeatureDiscoveryReconciler) updateStatus(nfd *nfdv1.NodeFeatureDiscovery, conditions []conditionsv1.Condition) error {

	// The actual 'nfd' object should *not* be modified when trying to
	// check the object's status. This variable is a dummy variable used
	// to set temporary conditions.
	nfdCopy := nfd.DeepCopy()

	nfdCopy.Status.Conditions = conditions

	//// If a set of conditions exists, then it should be added to the
	//// 'nfd' Copy.
	//if conditions != nil {
	//	nfdCopy.Status.Conditions = conditions
	//}

	//// Next step is to check if we need to update the status
	//modified := false
	//
	//// Because there are only four possible conditions (degraded, available,
	//// updatable, and progressing), it isn't necessary to check if old
	//// conditions should be removed.
	//for _, newCondition := range nfdCopy.Status.Conditions {
	//	oldCondition := conditionsv1.FindStatusCondition(nfd.Status.Conditions, newCondition.Type)
	//	if oldCondition == nil {
	//		modified = true
	//		break
	//	}
	//
	//	// Ignore timestamps to avoid infinite reconcile loops
	//	if oldCondition.Status != newCondition.Status ||
	//		oldCondition.Reason != newCondition.Reason ||
	//		oldCondition.Message != newCondition.Message {
	//
	//		modified = true
	//		break
	//	}
	//}

	//// If nothing has been modified, then return nothing. Even if the list
	//// of 'conditions' is not empty, it should not be counted as an update
	//// if it was already counted as an update before.
	//if !modified {
	//	return nil
	//}

	klog.Infof("Updating the NFD status")
	klog.Infof("Conditions: %v", conditions)
	return r.Status().Update(context.TODO(), nfdCopy)
}

// updateDegradedCondition is used to mark a given resource as "degraded" so that
// the reconciler can take steps to rectify the situation.
func (r *NodeFeatureDiscoveryReconciler) updateDegradedCondition(nfd *nfdv1.NodeFeatureDiscovery, condition string, conditionErr error) (ctrl.Result, error) {

	// It is already assumed that the resource has been degraded, so the first
	// step is to gather the correct list of conditions.
	conditions := r.getDegradedConditions(condition, conditionErr.Error())
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

func (r *NodeFeatureDiscoveryReconciler) getServiceAccountConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the service account
	sa, err := components.GetServiceAccount(nfd)
	if err != nil || sa == nil {
		messageString := fmt.Sprint(err)
		return r.getDegradedConditions(conditionReasonNFDDegraded, messageString), errors.New(conditionReasonNFDDegraded)
	}

	return nil, nil
}

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the role
	cr, err := components.GetClusterRole(nfd)
	if err != nil || cr == nil {
		messageString := fmt.Sprint(err)
		return r.getDegradedConditions(conditionReasonNFDDegraded, messageString), errors.New(conditionReasonNFDDegraded)
	}

	return nil, nil
}

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBindingConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the cluster role binding
	crb, err := components.GetClusterRoleBinding(nfd)
	if err != nil || crb == nil {
		messageString := fmt.Sprint(err)
		return r.getDegradedConditions(conditionReasonNFDDegraded, messageString), errors.New(conditionReasonNFDDegraded)
	}

	return nil, nil
}

func (r *NodeFeatureDiscoveryReconciler) getPodConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the pod
	pod, err := components.GetPod(nfd)
	if err != nil || pod == nil {
		messageString := fmt.Sprint(err)
		return r.getDegradedConditions(conditionReasonNFDDegraded, messageString), errors.New(conditionReasonNFDDegraded)
	}

	return nil, nil
}

func (r *NodeFeatureDiscoveryReconciler) getDaemonSetConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the daemon set binding
	ds, err := components.GetDaemonSet(nfd)
	if err != nil || ds == nil {
		messageString := fmt.Sprint(err)
		return r.getDegradedConditions(conditionReasonNFDDegraded, messageString), errors.New(conditionReasonNFDDegraded)
	}

	return nil, nil
}

func (r *NodeFeatureDiscoveryReconciler) getServiceConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get service
	svc, err := components.GetService(nfd)
	if err != nil || svc == nil {
		messageString := fmt.Sprint(err)
		return r.getDegradedConditions(conditionReasonNFDDegraded, messageString), errors.New(conditionReasonNFDDegraded)
	}

	return nil, nil
}

func (r *NodeFeatureDiscoveryReconciler) getWorkerConfigConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the worker config
	wc, err := components.GetWorkerConfig(nfd)
	if err != nil || wc == nil {
		messageString := fmt.Sprint(err)
		return r.getDegradedConditions(conditionReasonNFDDegraded, messageString), errors.New(conditionReasonNFDDegraded)
	}

	return nil, nil
}
