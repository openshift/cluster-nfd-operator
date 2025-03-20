/*
Copyright 2024 The Kubernetes Authors.

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
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nfdv1openshiftioalpha1 "github.com/openshift/cluster-nfd-operator/api/v1alpha1"
	nfdk8ssigsiov1alpha1 "github.com/openshift/cluster-nfd-operator/api/v1temp1"
	"github.com/openshift/cluster-nfd-operator/internal/client"
	"go.uber.org/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clt "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Reconcile", func() {
	var (
		ctrl *gomock.Controller
		clnt *client.MockClient
		nfr  *nodeFeatureRuleReconciler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)

		nfr = &nodeFeatureRuleReconciler{
			client: clnt,
			scheme: scheme,
		}
	})
	ctx := context.Background()

	It("Create NodeFeatureRule successfully", func() {
		nfdCR := nfdv1openshiftioalpha1.NodeFeatureRule{}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			clnt.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Return(nil),
		)

		res, err := nfr.Reconcile(ctx, &nfdCR)
		Expect(res).To(Equal(reconcile.Result{}))
		Expect(err).To(BeNil())
	})
	It("Fail to Create NodeFeatureRule", func() {
		nfdCR := nfdv1openshiftioalpha1.NodeFeatureRule{}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			clnt.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Return(fmt.Errorf("some error")),
		)
		res, err := nfr.Reconcile(ctx, &nfdCR)
		Expect(res).To(Equal(reconcile.Result{}))
		Expect(err).To(HaveOccurred())
	})

	It("Update NodeFeatureRule successfully", func() {
		nfdCR := nfdv1openshiftioalpha1.NodeFeatureRule{}
		nfdCRToUpdate := nfdk8ssigsiov1alpha1.NodeFeatureRule{
			Spec: nfdv1openshiftioalpha1.NodeFeatureRuleSpec{Rules: []nfdv1openshiftioalpha1.Rule{
				{Name: "test"}},
			},
		}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&nfdk8ssigsiov1alpha1.NodeFeatureRule{})).
				DoAndReturn(func(_ context.Context, _ clt.ObjectKey, obj clt.Object, opts ...clt.GetOption) error {
					target := obj.(*nfdk8ssigsiov1alpha1.NodeFeatureRule)
					*target = nfdCRToUpdate
					return nil
				}),
			clnt.EXPECT().Patch(ctx, gomock.Any(), gomock.Any()).Return(nil),
		)

		res, err := nfr.Reconcile(ctx, &nfdCR)
		Expect(res).To(Equal(reconcile.Result{}))
		Expect(err).To(BeNil())
	})
	It("Fail to Update NodeFeatureRule", func() {
		nfdCR := nfdv1openshiftioalpha1.NodeFeatureRule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-name",
			},
			Spec: nfdv1openshiftioalpha1.NodeFeatureRuleSpec{
				Rules: []nfdv1openshiftioalpha1.Rule{{Name: "rule-test-name"}},
			},
		}
		nfdCRToUpdate := nfdk8ssigsiov1alpha1.NodeFeatureRule{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-name",
			},
			Spec: nfdv1openshiftioalpha1.NodeFeatureRuleSpec{},
		}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.AssignableToTypeOf(&nfdk8ssigsiov1alpha1.NodeFeatureRule{})).
				DoAndReturn(func(_ context.Context, _ clt.ObjectKey, obj clt.Object, opts ...clt.GetOption) error {
					target := obj.(*nfdk8ssigsiov1alpha1.NodeFeatureRule)
					*target = nfdCRToUpdate
					return nil
				}),
			clnt.EXPECT().Patch(ctx, gomock.Any(), gomock.Any()).Return(fmt.Errorf("some error")),
		)

		res, err := nfr.Reconcile(ctx, &nfdCR)
		Expect(res).To(Equal(reconcile.Result{}))
		Expect(err).To(HaveOccurred())
	})
})
