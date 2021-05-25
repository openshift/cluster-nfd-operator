package controllers

import (
	"bytes"
	"context"
	"time"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"github.com/openshift/cluster-nfd-operator/pkq/controller/nodefeaturediscovery/components"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
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

	// Condition references
	conditionAvailable   = conditionsv1.ConditionAvailable
	conditionUpgradeable = conditionsv1.ConditionUpgradeable
	conditionDegraded    = conditionsv1.ConditionDegraded
	conditionProgressing = conditionsv1.ConditionProgressing
)

func (r *NodeFeatureDiscoveryReconciler) updateStatus(nfd *nfdv1.NodeFeatureDiscovery, conditions []conditionsv1.Condition) error {
	nfdCopy := nfd.DeepCopy()

	if conditions != nil {
		nfdCopy.Status.Conditions = conditions
	}

	// check if we need to update the status
	modified := false

	// since we always set the same four conditions, we don't need to check if we need to remove old conditions
	for _, newCondition := range nfdCopy.Status.Conditions {
		oldCondition := conditionsv1.FindStatusCondition(nfd.Status.Conditions, newCondition.Type)
		if oldCondition == nil {
			modified = true
			break
		}

		// ignore timestamps to avoid infinite reconcile loops
		if oldCondition.Status != newCondition.Status ||
			oldCondition.Reason != newCondition.Reason ||
			oldCondition.Message != newCondition.Message {

			modified = true
			break
		}
	}

	if !modified {
		return nil
	}

	klog.Infof("Updating the Node Feature Discovery resources status")
	return r.Status().Update(context.TODO(), nfdCopy)
}

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
	nfdServiceAccount, err := components.GetServiceAccount(nfd)
	if err != nil {
		return nil, err
	}

	// This variable will keep track of the degraded condition
	var degradedServiceAccountCondition conditionsv1.Condition

	// Message strings to print out status
	var messageString string

	// We need to iterate through all the service account's conditions, and keep track if any of
	// them are listed as being degraded
	isDegraded := false
	for _, condition := range nfdServiceAccount.Status.Conditions {
		if (condition.Type == conditionDegraded) && condition.Status == corev1.ConditionTrue {
			isDegraded = true
			degradedServiceAccountCondition = &condition
		}
	}

	// If the condition status is listed as "degraded", write the reason and message explaining why
	if isDegraded == true {
		//if len(degradedServiceAccountCondition.Reason) > 0 {
		//	message.WriteString("ServiceAccount " + sa.GetName() + " Degraded Reason: " + degradedServiceAccountCondition.Reason + ".\n")
		//}
		//if len(degradedServiceAccountCondition.Message) > 0 {
		//	message.WriteString("ServiceAccount " + sa.GetName() + " Degraded Message: " + degradedServiceAccountCondition.Message + ".\n")
		//}
		messageString = setMessageString("ServiceAccount", nfdServiceAccount.GetName(), degradedServiceAccountCondition)
	}

	// If we have no message, then everything looks good
	if len(messageString) == 0 {
		return nil, nil
	}

	return r.getDegradedConditions(degradedServiceAccountCondition, messageString), nil
}

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the role
	nfdClusterRole, err := components.GetClusterRole(nfd)
	if err != nil {
		return nil, err
	}

	// This variable will keep track of the degraded condition
	var degradedClusterRoleCondition conditionsv1.Condition

	// Message string to print out status
	var messageString string

	// We need to iterate through all the cluster role's conditions, and keep track if
	// any of them are listed as being degraded
	isDegraded := false
	for _, condition := range nfdClusterRole.Status.Conditions {
		if (condition.Type == conditionDegraded) && condition.Status == corev1.ConditionTrue {
			isDegraded = true
			degradedClusterRoleCondition = &condition
		}
	}

	// If the condition status is listed as "degraded", write the reason and message explaining why
	if isDegraded == true {
		//if len(degradedClusterRoleCondition.Reason) > 0 {
		//	message.WriteString("ServiceAccount " + nfdClusterRole.GetName() + " Degraded Reason: " + degradedClusterRoleCondition.Reason + ".\n")
		//}
		//if len(degradedServiceAccountCondition.Message) > 0 {
		//	message.WriteString("ServiceAccount " + nfdClusterRole.GetName() + " Degraded Message: " + degradedClusterRoleCondition.Message + ".\n")
		//}
		messageString = setMessageString("ClusterRole", nfdClusterRole.GetName(), degradedClusterRoleCondition)
	}

	// If we have no message, then everything looks good
	if len(messageString) == 0 {
		return nil, nil
	}

	return r.getDegradedConditions(degradedClusterRoleCondition, messageString), nil
}

func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBindingConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the cluster role binding
	nfdClusterRoleBinding, err := components.GetClusterRoleBinding(nfd)
	if err != nil {
		return nil, err
	}

	// This variable will keep track of the degraded condition
	var degradedClusterRoleBindingCondition conditionsv1.Condition

	// Message string to print out status
	var messageString string

	// We need to iterate through all the cluster role's conditions, and keep track if any of them
	// are listed as being degraded
	isDegraded := false
	for _, condition := range nfdClusterRoleBinding.Status.Conditions {
		if (condition.Type == conditionDegraded) && condition.Status == corev1.ConditionTrue {
			isDegraded = true
			degradedClusterRoleBindingCondition = &condition
		}
	}

	// If the condition status is listed as "degraded", write the reason and message explaining why
	if isDegraded == true {
		//if len(degradedClusterRoleBindingCondition.Reason) > 0 {
		//	message.WriteString("RoleBinding " + nfdClusterRoleBinding.GetName() + " Degraded Reason: " + degradedClusterRoleBindingCondition.Reason + ".\n")
		//}
		//if len(degradedServiceAccountCondition.Message) > 0 {
		//	message.WriteString("RoleBinding " + nfdClusterRoleBinding.GetName() + " Degraded Message: " + degradedClusterRoleBindingCondition.Message + ".\n")
		//}
		messageString = setMessageString("RoleBinding", nfdClusterRoleBinding.GetName(), degradedClusterRoleBindingCondition)
	}

	// If we have no message, then everything looks good
	if len(messageString) == 0 {
		return nil, nil
	}

	return r.getDegradedConditions(degradedClusterRoleBindingCondition, messageString), nil
}

func (r *NodeFeatureDiscoveryReconciler) getDaemonSetConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the daemon set binding
	nfdDaemonSet, err := components.GetDaemonSet(nfd)
	if err != nil {
		return nil, err
	}

	// This variable will keep track of the degraded condition
	var degradedDaemonSetCondition conditionsv1.Condition

	// Message string to print out status
	var messageString string

	// We need to iterate through all the daemon set's conditions, and keep track if any of them
	// are listed as being degraded
	isDegraded := false
	for _, condition := range nfdDaemonSet.Status.Conditions {
		if (condition.Type == conditionDegraded) && condition.Status == corev1.ConditionTrue {
			isDegraded = true
			degradedDaemonSetCondition = &condition
		}
	}

	// If the condition status is listed as "degraded", write the reason and message explaining why
	if isDegraded == true {
		//if len(degradedRoleBindingCondition.Reason) > 0 {
		//	message.WriteString("DaemonSet " + nfdDaemonSet.GetName() + " Degraded Reason: " + degradedRoleBindingCondition.Reason + ".\n")
		//}
		//if len(degradedServiceAccountCondition.Message) > 0 {
		//	message.WriteString("DaemonSet " + nfdDaemonSet.GetName() + " Degraded Message: " + degradedRoleBindingCondition.Message + ".\n")
		//}
		messageString = setMessageString("DaemonSet", nfdDaemonSet.GetName(), degradedDaemonSetCondition)

		// If we have no message, then everything looks good
	}

	if len(messageString) == 0 {
		return nil, nil
	}

	return r.getDegradedConditions(degradedDaemonSetCondition, messageString), nil
}

func (r *NodeFeatureDiscoveryReconciler) getServiceConditions(nfd *nfdv1.NodeFeatureDiscovery) ([]conditionsv1.Condition, error) {

	// Attempt to get the daemon set binding
	nfdService, err := components.GetService(nfd)
	if err != nil {
		return nil, err
	}

	// This variable will keep track of the degraded condition
	var degradedServiceCondition conditionsv1.Condition

	// Message string to print out status
	var messageString string

	// We need to iterate through all the service's conditions, and keep track if any of them
	// are listed as being degraded
	isDegraded := false
	for _, condition := range nfdService.Status.Conditions {
		if (condition.Type == conditionDegraded) && condition.Status == corev1.ConditionTrue {
			isDegraded = true
			degradedServiceCondition = &condition
		}
	}

	// If the condition status is listed as "degraded", write the reason and message explaining why
	if isDegraded == true {
		//if len(degradedRoleBindingCondition.Reason) > 0 {
		//	message.WriteString("Service " + nfdService.GetName() + " Degraded Reason: " + degradedServiceCondition.Reason + ".\n")
		//}
		//if len(degradedServiceAccountCondition.Message) > 0 {
		//	message.WriteString("Service " + nfdService.GetName() + " Degraded Message: " + degradedServiceCondition.Message + ".\n")
		//}
		messageString = setMessageString("Service", nfdService.GetName(), degradedServiceCondition)

	}

	// If we have no message, then everything looks good
	if len(messageString) == 0 {
		return nil, nil
	}

	return r.getDegradedConditions(degradedServiceCondition, messageString), nil
}

// Sets error message if the resource is degraded
func setMessageString(clusterComponentType string, clusterComponentName string, degradedCondition conditionsv1.Condition) string {

	// Initialize the message. This can stay empty if nothing is wrong.
	message := bytes.Buffer{}

	if len(degradedCondition.Reason) > 0 {
		message.WriteString(clusterComponentType + " " + clusterComponentName + " Degraded Reason: " + degradedCondition.Reason + ".\n")
	}
	if len(degradedCondition.Message) > 0 {
		message.WriteString(clusterComponentType + " " + clusterComponentName + " Degraded Message: " + degradedCondition.Message + ".\n")
	}

	// Convert to a string and check if there is any message
	messageString := message.String()
	return messageString
}
