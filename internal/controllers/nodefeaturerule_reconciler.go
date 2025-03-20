/*
Copyright 2021. The Kubernetes Authors.

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

package new_controllers

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nfdopenshiftiov1alpha1 "github.com/openshift/cluster-nfd-operator/api/v1alpha1"
	nfdk8ssigsiov1alpha1 "github.com/openshift/cluster-nfd-operator/api/v1temp1"
)

// NodeFeatureRuleReconciler reconciles a NodeFeatureRule object
type nodeFeatureRuleReconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

func NewNodeFeatureRuleReconciler(client client.Client, scheme *runtime.Scheme) *nodeFeatureRuleReconciler {
	return &nodeFeatureRuleReconciler{
		client: client,
		scheme: scheme,
	}
}

// +kubebuilder:rbac:groups=nfd.openshift.io,resources=nodefeaturerules,verbs=get;list;watch
// +kubebuilder:rbac:groups=nfd.k8s-sigs.io,resources=nodefeaturerules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nfd.openshift.io,resources=nodefeaturerules/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the NodeFeatureRule object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *nodeFeatureRuleReconciler) Reconcile(ctx context.Context, nfr *nfdopenshiftiov1alpha1.NodeFeatureRule) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling NodeFeatureRule from nfd.openshift.io group", "name", nfr.Name)

	target := &nfdk8ssigsiov1alpha1.NodeFeatureRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfr.Name,
			Namespace: nfr.Namespace,
		},
	}

	result, err := controllerutil.CreateOrPatch(ctx, r.client, target, func() error {
		if err := controllerutil.SetControllerReference(nfr, target, r.scheme); err != nil {
			return err
		}
		target.Spec = nfdopenshiftiov1alpha1.NodeFeatureRuleSpec{
			Rules: nfr.Spec.Rules,
		}
		return nil
	})

	if err != nil {
		logger.Error(err, "Failed to create or update NodeFeatureRule in nfd.k8s-sigs.io group")
		return ctrl.Result{}, err
	}

	switch result {
	case controllerutil.OperationResultCreated:
		logger.Info("Successfully created NodeFeatureRule in nfd.k8s-sigs.io group")
	case controllerutil.OperationResultUpdated:
		logger.Info("Successfully updated NodeFeatureRule in nfd.k8s-sigs.io group")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *nodeFeatureRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nfdopenshiftiov1alpha1.NodeFeatureRule{}).
		Complete(reconcile.AsReconciler[*nfdopenshiftiov1alpha1.NodeFeatureRule](mgr.GetClient(), r))
}
