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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openshift/cluster-nfd-operator/internal/client"
	"github.com/openshift/cluster-nfd-operator/internal/configmap"
	"github.com/openshift/cluster-nfd-operator/internal/daemonset"
	"github.com/openshift/cluster-nfd-operator/internal/deployment"
	"github.com/openshift/cluster-nfd-operator/internal/job"
	"github.com/openshift/cluster-nfd-operator/internal/scc"
	"github.com/openshift/cluster-nfd-operator/internal/status"
)

var _ = Describe("Reconcile", func() {
	var (
		ctrl       *gomock.Controller
		mockHelper *MocknodeFeatureDiscoveryHelperAPI
		nfdr       *nodeFeatureDiscoveryReconciler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockHelper = NewMocknodeFeatureDiscoveryHelperAPI(ctrl)

		nfdr = &nodeFeatureDiscoveryReconciler{
			helper: mockHelper,
		}
	})

	ctx := context.Background()

	It("good flow without finalization", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}

		mockHelper.EXPECT().hasFinalizer(&nfdCR).Return(true)
		mockHelper.EXPECT().handleSCCs(ctx, &nfdCR).Return(nil)
		mockHelper.EXPECT().handleMaster(ctx, &nfdCR, nfdCR.Spec.Operand.Image).Return(nil)
		mockHelper.EXPECT().handleWorker(ctx, &nfdCR, nfdCR.Spec.Operand.Image).Return(nil)
		mockHelper.EXPECT().handleTopology(ctx, &nfdCR, nfdCR.Spec.Operand.Image).Return(nil)
		mockHelper.EXPECT().handleGC(ctx, &nfdCR, nfdCR.Spec.Operand.Image).Return(nil)
		mockHelper.EXPECT().handleStatus(ctx, &nfdCR).Return(nil)

		res, err := nfdr.Reconcile(ctx, &nfdCR)
		Expect(res).To(Equal(reconcile.Result{}))
		Expect(err).To(BeNil())
	})

	DescribeTable("finalization flow", func(finalizeComponentsError, handlePruneError, pruneDone, removeFinalizerError bool) {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		timestamp := metav1.Now()
		nfdCR.SetDeletionTimestamp(&timestamp)

		if finalizeComponentsError {
			mockHelper.EXPECT().finalizeComponents(ctx, &nfdCR).Return(fmt.Errorf("some error"))
			goto executeTestFunction
		}
		mockHelper.EXPECT().finalizeComponents(ctx, &nfdCR).Return(nil)
		if handlePruneError {
			mockHelper.EXPECT().handlePrune(ctx, &nfdCR).Return(false, fmt.Errorf("some error"))
			goto executeTestFunction
		}
		if !pruneDone {
			mockHelper.EXPECT().handlePrune(ctx, &nfdCR).Return(false, nil)
			goto executeTestFunction
		}
		mockHelper.EXPECT().handlePrune(ctx, &nfdCR).Return(true, nil)
		if removeFinalizerError {
			mockHelper.EXPECT().removeFinalizer(ctx, &nfdCR).Return(fmt.Errorf("some error"))
			goto executeTestFunction
		}
		mockHelper.EXPECT().removeFinalizer(ctx, &nfdCR).Return(nil)

	executeTestFunction:

		res, err := nfdr.Reconcile(ctx, &nfdCR)
		Expect(res).To(Equal(reconcile.Result{}))
		if finalizeComponentsError || handlePruneError || removeFinalizerError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).To(BeNil())
		}
	},
		Entry("finalizeComponents failed", true, false, false, false),
		Entry("handlePrune failed", false, true, false, false),
		Entry("handlePrune succeeded but not done yet", false, false, false, false),
		Entry("handlePrune succeeded and done, removeFinalizer failed", false, false, true, true),
		Entry("fully successfull flow", false, false, true, false),
	)

	DescribeTable("setFinalizer flow", func(setFinalizerError error) {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		mockHelper.EXPECT().hasFinalizer(&nfdCR).Return(false)
		mockHelper.EXPECT().setFinalizer(ctx, &nfdCR).Return(setFinalizerError)

		res, err := nfdr.Reconcile(ctx, &nfdCR)
		Expect(res).To(Equal(reconcile.Result{}))
		if setFinalizerError != nil {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).To(BeNil())
		}
	},
		Entry("setFinalizer failed", fmt.Errorf("set finalizer error")),
		Entry("setFinalizer succeeded", fmt.Errorf("set finalizer error")),
	)

	DescribeTable("check components error flows", func(handlerSCCError,
		handlerMasterError,
		handlerWorkerError,
		handleTopologyError,
		handlerGCError,
		handlePruneError,
		handleStatusError error) {
		nfdCR := nfdv1.NodeFeatureDiscovery{}

		mockHelper.EXPECT().hasFinalizer(&nfdCR).Return(true)
		mockHelper.EXPECT().handleSCCs(ctx, &nfdCR).Return(handlerSCCError)
		mockHelper.EXPECT().handleMaster(ctx, &nfdCR, nfdCR.Spec.Operand.Image).Return(handlerMasterError)
		mockHelper.EXPECT().handleWorker(ctx, &nfdCR, nfdCR.Spec.Operand.Image).Return(handlerWorkerError)
		mockHelper.EXPECT().handleTopology(ctx, &nfdCR, nfdCR.Spec.Operand.Image).Return(handleTopologyError)
		mockHelper.EXPECT().handleGC(ctx, &nfdCR, nfdCR.Spec.Operand.Image).Return(handlerGCError)
		mockHelper.EXPECT().handleStatus(ctx, &nfdCR).Return(handleStatusError)

		res, err := nfdr.Reconcile(ctx, &nfdCR)
		Expect(res).To(Equal(reconcile.Result{}))
		if handlerSCCError != nil || handlerMasterError != nil || handlerWorkerError != nil || handleTopologyError != nil ||
			handlerGCError != nil || handlePruneError != nil || handleStatusError != nil {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).To(BeNil())
		}
	},
		Entry("handleSCCs failed", fmt.Errorf("scc error"), nil, nil, nil, nil, nil, nil),
		Entry("handleMaster failed", nil, fmt.Errorf("master error"), nil, nil, nil, nil, nil),
		Entry("handleWorker failed", nil, nil, fmt.Errorf("worker error"), nil, nil, nil, nil),
		Entry("handleTopology failed", nil, nil, nil, fmt.Errorf("topology error"), nil, nil, nil),
		Entry("handleGC failed", nil, nil, nil, nil, fmt.Errorf("gc error"), nil, nil),
		Entry("handleStatus failed", nil, nil, nil, nil, nil, nil, fmt.Errorf("status error")),
		Entry("all components succeeded", nil, nil, nil, nil, nil, nil, nil),
	)
})

var _ = Describe("handleMaster", func() {
	var (
		ctrl           *gomock.Controller
		clnt           *client.MockClient
		mockDeployment *deployment.MockDeploymentAPI
		nfdh           nodeFeatureDiscoveryHelperAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)
		mockDeployment = deployment.NewMockDeploymentAPI(ctrl)

		nfdh = newNodeFeatureDiscoveryHelperAPI(clnt, mockDeployment, nil, nil, nil, nil, nil, scheme)
	})

	ctx := context.Background()

	It("should create new nfd-master deployment if it does not exist", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockDeployment.EXPECT().SetMasterDeploymentAsDesired(&nfdCR, gomock.Any(), nfdCR.Spec.Operand.Image).Return(nil),
			clnt.EXPECT().Create(ctx, gomock.Any()).Return(nil),
		)

		err := nfdh.handleMaster(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(BeNil())
	})

	It("deployment exists, no need to create it, update is not executed", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nfd-cr",
				Namespace: "test-namespace",
			},
		}
		existingDeployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: nfdCR.Namespace, Name: "nfd-master"},
		}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ interface{}, _ interface{}, dp *appsv1.Deployment, _ ...ctrlclient.GetOption) error {
					dp.SetName(existingDeployment.Name)
					dp.SetNamespace(existingDeployment.Namespace)
					return nil
				},
			),
			mockDeployment.EXPECT().SetMasterDeploymentAsDesired(&nfdCR, &existingDeployment, nfdCR.Spec.Operand.Image).Return(nil),
		)

		err := nfdh.handleMaster(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(BeNil())
	})

	It("error flow, failed to populate deployment object", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockDeployment.EXPECT().SetMasterDeploymentAsDesired(&nfdCR, gomock.Any(), nfdCR.Spec.Operand.Image).Return(fmt.Errorf("some error")),
		)

		err := nfdh.handleMaster(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("handleWorker", func() {
	var (
		ctrl   *gomock.Controller
		clnt   *client.MockClient
		mockDS *daemonset.MockDaemonsetAPI
		mockCM *configmap.MockConfigMapAPI
		nfdh   nodeFeatureDiscoveryHelperAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)
		mockDS = daemonset.NewMockDaemonsetAPI(ctrl)
		mockCM = configmap.NewMockConfigMapAPI(ctrl)

		nfdh = newNodeFeatureDiscoveryHelperAPI(clnt, nil, mockDS, mockCM, nil, nil, nil, scheme)
	})

	ctx := context.Background()
	nfdCR := nfdv1.NodeFeatureDiscovery{}

	It("both configmap and daemonset are missing, they should both be created", func() {
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockCM.EXPECT().SetWorkerConfigMapAsDesired(ctx, &nfdCR, gomock.Any()).Return(nil),
			clnt.EXPECT().Create(ctx, gomock.Any()).Return(nil),
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockDS.EXPECT().SetWorkerDaemonsetAsDesired(ctx, &nfdCR, gomock.Any(), nfdCR.Spec.Operand.Image).Return(nil),
			clnt.EXPECT().Create(ctx, gomock.Any()).Return(nil),
		)

		err := nfdh.handleWorker(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(BeNil())
	})

	It("worker config and daemonset exist, no need to create them, update is not executed", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nfd-cr",
				Namespace: "test-namespace",
			},
			Spec: nfdv1.NodeFeatureDiscoverySpec{
				TopologyUpdater: true,
			},
		}
		existingDS := appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Namespace: nfdCR.Namespace, Name: "nfd-worker"},
		}
		existingCM := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: nfdCR.Namespace, Name: "nfd-worker"},
		}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ interface{}, _ interface{}, cm *corev1.ConfigMap, _ ...ctrlclient.GetOption) error {
					cm.SetName(existingCM.Name)
					cm.SetNamespace(existingCM.Namespace)
					return nil
				},
			),
			mockCM.EXPECT().SetWorkerConfigMapAsDesired(ctx, &nfdCR, &existingCM).Return(nil),
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ interface{}, _ interface{}, ds *appsv1.DaemonSet, _ ...ctrlclient.GetOption) error {
					ds.SetName(existingDS.Name)
					ds.SetNamespace(existingDS.Namespace)
					return nil
				},
			),
			mockDS.EXPECT().SetWorkerDaemonsetAsDesired(ctx, &nfdCR, &existingDS, nfdCR.Spec.Operand.Image).Return(nil),
		)

		err := nfdh.handleWorker(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(BeNil())
	})

	It("error flow, failed to populate configmap object", func() {
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockCM.EXPECT().SetWorkerConfigMapAsDesired(ctx, &nfdCR, gomock.Any()).Return(fmt.Errorf("some error")),
		)

		err := nfdh.handleWorker(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(HaveOccurred())
	})

	It("error flow, failed to populate daemonset object", func() {
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockCM.EXPECT().SetWorkerConfigMapAsDesired(ctx, &nfdCR, gomock.Any()).Return(nil),
			clnt.EXPECT().Create(ctx, gomock.Any()).Return(nil),
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockDS.EXPECT().SetWorkerDaemonsetAsDesired(ctx, &nfdCR, gomock.Any(), nfdCR.Spec.Operand.Image).Return(fmt.Errorf("some error")),
		)

		err := nfdh.handleWorker(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("handleTopology", func() {
	var (
		ctrl   *gomock.Controller
		clnt   *client.MockClient
		mockDS *daemonset.MockDaemonsetAPI
		nfdh   nodeFeatureDiscoveryHelperAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)
		mockDS = daemonset.NewMockDaemonsetAPI(ctrl)

		nfdh = newNodeFeatureDiscoveryHelperAPI(clnt, nil, mockDS, nil, nil, nil, nil, scheme)
	})

	ctx := context.Background()

	It("should create new nfd-topology daemonset if it does not exist", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{
			Spec: nfdv1.NodeFeatureDiscoverySpec{
				TopologyUpdater: true,
			},
		}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockDS.EXPECT().SetTopologyDaemonsetAsDesired(ctx, &nfdCR, gomock.Any(), nfdCR.Spec.Operand.Image).Return(nil),
			clnt.EXPECT().Create(ctx, gomock.Any()).Return(nil),
		)

		err := nfdh.handleTopology(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(BeNil())
	})

	It("topology daemonset exists, no need to create it, update is not executed", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nfd-cr",
				Namespace: "test-namespace",
			},
			Spec: nfdv1.NodeFeatureDiscoverySpec{
				TopologyUpdater: true,
			},
		}
		existingDS := appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Namespace: nfdCR.Namespace, Name: "nfd-topology-updater"},
		}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ interface{}, _ interface{}, ds *appsv1.DaemonSet, _ ...ctrlclient.GetOption) error {
					ds.SetName(existingDS.Name)
					ds.SetNamespace(existingDS.Namespace)
					return nil
				},
			),
			mockDS.EXPECT().SetTopologyDaemonsetAsDesired(ctx, &nfdCR, &existingDS, nfdCR.Spec.Operand.Image).Return(nil),
		)

		err := nfdh.handleTopology(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(BeNil())
	})

	It("error flow, failed to populate daemonset object", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{
			Spec: nfdv1.NodeFeatureDiscoverySpec{
				TopologyUpdater: true,
			},
		}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockDS.EXPECT().SetTopologyDaemonsetAsDesired(ctx, &nfdCR, gomock.Any(), nfdCR.Spec.Operand.Image).Return(fmt.Errorf("some error")),
		)

		err := nfdh.handleTopology(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(HaveOccurred())
	})

	It("if TopologyUpdate not set - nothing to do", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}

		err := nfdh.handleTopology(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(BeNil())
	})
})

var _ = Describe("handleGC", func() {
	var (
		ctrl           *gomock.Controller
		clnt           *client.MockClient
		mockDeployment *deployment.MockDeploymentAPI
		nfdh           nodeFeatureDiscoveryHelperAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)
		mockDeployment = deployment.NewMockDeploymentAPI(ctrl)

		nfdh = newNodeFeatureDiscoveryHelperAPI(clnt, mockDeployment, nil, nil, nil, nil, nil, scheme)
	})

	ctx := context.Background()

	It("should create new nfd-gc deployment if it does not exist", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockDeployment.EXPECT().SetGCDeploymentAsDesired(&nfdCR, gomock.Any(), nfdCR.Spec.Operand.Image).Return(nil),
			clnt.EXPECT().Create(ctx, gomock.Any()).Return(nil),
		)

		err := nfdh.handleGC(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(BeNil())
	})

	It("nfd-gc deployment exists, no need to create it, update is not executed", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nfd-cr",
				Namespace: "test-namespace",
			},
		}
		existingDeployment := appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: nfdCR.Namespace, Name: "nfd-gc"},
		}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ interface{}, _ interface{}, dp *appsv1.Deployment, _ ...ctrlclient.GetOption) error {
					dp.SetName(existingDeployment.Name)
					dp.SetNamespace(existingDeployment.Namespace)
					return nil
				},
			),
			mockDeployment.EXPECT().SetGCDeploymentAsDesired(&nfdCR, &existingDeployment, nfdCR.Spec.Operand.Image).Return(nil),
		)

		err := nfdh.handleGC(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(BeNil())
	})

	It("error flow, failed to populate nfd-gc deployment object", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		gomock.InOrder(
			clnt.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).Return(apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockDeployment.EXPECT().SetGCDeploymentAsDesired(&nfdCR, gomock.Any(), nfdCR.Spec.Operand.Image).Return(fmt.Errorf("some error")),
		)

		err := nfdh.handleGC(ctx, &nfdCR, nfdCR.Spec.Operand.Image)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("hasFinalizer", func() {
	It("checking return status whether finalizer set or not", func() {
		nfdh := newNodeFeatureDiscoveryHelperAPI(nil, nil, nil, nil, nil, nil, nil, nil)

		By("finalizers was empty")
		nfdCR := nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "nfd-cr",
				Finalizers: nil,
			},
		}
		res := nfdh.hasFinalizer(&nfdCR)
		Expect(res).To(BeFalse())

		By("finalizers exists, but NFD finalizer missing")
		nfdCR = nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "nfd-cr",
				Finalizers: []string{"some finalizer"},
			},
		}
		res = nfdh.hasFinalizer(&nfdCR)
		Expect(res).To(BeFalse())

		By("finalizers exists, but NFD finalizer present")
		nfdCR = nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "nfd-cr",
				Finalizers: []string{"some finalizer", finalizerLabel},
			},
		}
		res = nfdh.hasFinalizer(&nfdCR)
		Expect(res).To(BeTrue())
	})
})

var _ = Describe("setFinalizer", func() {
	var (
		ctrl *gomock.Controller
		clnt *client.MockClient
		nfdh nodeFeatureDiscoveryHelperAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)
		nfdh = newNodeFeatureDiscoveryHelperAPI(clnt, nil, nil, nil, nil, nil, nil, nil)
	})

	It("checking the return status of setFinalizer function", func() {
		ctx := context.Background()

		By("Updating the NFD instance fails, original finalizers was empty")
		nfdCR := nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "nfd-cr",
				Finalizers: nil,
			},
		}
		expectedCR := nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "nfd-cr",
				Finalizers: []string{finalizerLabel},
			},
		}
		clnt.EXPECT().Update(ctx, &expectedCR).Return(fmt.Errorf("some error"))
		err := nfdh.setFinalizer(ctx, &nfdCR)
		Expect(err).ToNot(BeNil())

		By("Updating the NFD instance succeeds")
		nfdCR = nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "nfd-cr",
				Finalizers: []string{"some finalizer"},
			},
		}
		expectedCR = nfdv1.NodeFeatureDiscovery{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "nfd-cr",
				Finalizers: []string{"some finalizer", finalizerLabel},
			},
		}
		clnt.EXPECT().Update(ctx, &expectedCR).Return(nil)
		err = nfdh.setFinalizer(ctx, &nfdCR)
		Expect(err).To(BeNil())
	})
})

var _ = Describe("finalizeComponents", func() {
	var (
		ctrl           *gomock.Controller
		clnt           *client.MockClient
		mockDeployment *deployment.MockDeploymentAPI
		mockDS         *daemonset.MockDaemonsetAPI
		mockCM         *configmap.MockConfigMapAPI
		mockSCC        *scc.MockSccAPI
		nfdh           nodeFeatureDiscoveryHelperAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)
		mockDeployment = deployment.NewMockDeploymentAPI(ctrl)
		mockDS = daemonset.NewMockDaemonsetAPI(ctrl)
		mockCM = configmap.NewMockConfigMapAPI(ctrl)
		mockSCC = scc.NewMockSccAPI(ctrl)

		nfdh = newNodeFeatureDiscoveryHelperAPI(clnt, mockDeployment, mockDS, mockCM, nil, mockSCC, nil, scheme)
	})

	ctx := context.Background()
	namespace := "test-namespace"
	nfdCR := nfdv1.NodeFeatureDiscovery{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		Spec: nfdv1.NodeFeatureDiscoverySpec{
			TopologyUpdater: true,
		},
	}

	DescribeTable("check finalization normal and error flows", func(deleteWorkerDSError,
		deleteWorkerCMError,
		deleteTopologyDSError,
		deleteMasterDeploymentError,
		deleteGCDeploymentError,
		deleteWorkerSCCError,
		deleteTopologySCCError bool) {

		if deleteWorkerDSError {
			mockDS.EXPECT().DeleteDaemonSet(ctx, namespace, "nfd-worker").Return(fmt.Errorf("some error"))
			goto executeTestFunction
		}
		mockDS.EXPECT().DeleteDaemonSet(ctx, namespace, "nfd-worker").Return(nil)
		if deleteWorkerCMError {
			mockCM.EXPECT().DeleteConfigMap(ctx, namespace, "nfd-worker").Return(fmt.Errorf("some error"))
			goto executeTestFunction
		}
		mockCM.EXPECT().DeleteConfigMap(ctx, namespace, "nfd-worker").Return(nil)
		if deleteTopologyDSError {
			mockDS.EXPECT().DeleteDaemonSet(ctx, namespace, "nfd-topology-updater").Return(fmt.Errorf("some error"))
			goto executeTestFunction
		}
		mockDS.EXPECT().DeleteDaemonSet(ctx, namespace, "nfd-topology-updater").Return(nil)
		if deleteMasterDeploymentError {
			mockDeployment.EXPECT().DeleteDeployment(ctx, namespace, "nfd-master").Return(fmt.Errorf("some error"))
			goto executeTestFunction
		}
		mockDeployment.EXPECT().DeleteDeployment(ctx, namespace, "nfd-master").Return(nil)
		if deleteGCDeploymentError {
			mockDeployment.EXPECT().DeleteDeployment(ctx, namespace, "nfd-gc").Return(fmt.Errorf("some error"))
			goto executeTestFunction
		}
		mockDeployment.EXPECT().DeleteDeployment(ctx, namespace, "nfd-gc").Return(nil)
		if deleteWorkerSCCError {
			mockSCC.EXPECT().DeleteSCC(ctx, "nfd-worker").Return(fmt.Errorf("some error"))
			goto executeTestFunction
		}
		mockSCC.EXPECT().DeleteSCC(ctx, "nfd-worker").Return(nil)
		if deleteTopologySCCError {
			mockSCC.EXPECT().DeleteSCC(ctx, "nfd-topology-updater").Return(fmt.Errorf("some error"))
			goto executeTestFunction
		}
		mockSCC.EXPECT().DeleteSCC(ctx, "nfd-topology-updater").Return(nil)

	executeTestFunction:

		err := nfdh.finalizeComponents(ctx, &nfdCR)

		if deleteGCDeploymentError || deleteWorkerDSError || deleteWorkerCMError ||
			deleteTopologyDSError || deleteMasterDeploymentError || deleteWorkerSCCError || deleteTopologySCCError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).To(BeNil())
		}
	},
		Entry("delete worker daemonset failed", true, false, false, false, false, false, false),
		Entry("delete worker configmap failed", false, true, false, false, false, false, false),
		Entry("delete topology daemonset failed", false, false, true, false, false, false, false),
		Entry("delete master deployment failed", false, false, false, true, false, false, false),
		Entry("delete gc deployment failed", false, false, false, false, true, false, false),
		Entry("delete worker scc  failed", false, false, false, false, false, true, false),
		Entry("delete topology scc  failed", false, false, false, false, false, false, true),
		Entry("finalization flow was succesful", false, false, false, false, false, false, false),
	)
})

var _ = Describe("removeFinalizer", func() {
	var (
		ctrl *gomock.Controller
		clnt *client.MockClient
		nfdh nodeFeatureDiscoveryHelperAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)

		nfdh = newNodeFeatureDiscoveryHelperAPI(clnt, nil, nil, nil, nil, nil, nil, scheme)
	})

	ctx := context.Background()

	It("removing existing finalizer", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		controllerutil.AddFinalizer(&nfdCR, finalizerLabel)
		clnt.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		err := nfdh.removeFinalizer(ctx, &nfdCR)

		Expect(err).To(BeNil())
	})

	It("removing existing finalizer failed", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}
		controllerutil.AddFinalizer(&nfdCR, finalizerLabel)
		clnt.EXPECT().Update(ctx, gomock.Any()).Return(fmt.Errorf("some error"))

		err := nfdh.removeFinalizer(ctx, &nfdCR)

		Expect(err).To(HaveOccurred())
	})

	It("removing non-existing finalizer", func() {
		nfdCR := nfdv1.NodeFeatureDiscovery{}

		err := nfdh.removeFinalizer(ctx, &nfdCR)

		Expect(err).To(BeNil())
	})
})

var _ = Describe("handlePrune", func() {
	var (
		ctrl    *gomock.Controller
		mockJob *job.MockJobAPI
		nfdh    nodeFeatureDiscoveryHelperAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockJob = job.NewMockJobAPI(ctrl)
		nfdh = newNodeFeatureDiscoveryHelperAPI(nil, nil, nil, nil, mockJob, nil, nil, scheme)
	})

	ctx := context.Background()
	namespace := "test-namespace"
	nfdCR := nfdv1.NodeFeatureDiscovery{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
	}

	It("prune not defined in the CR", func() {
		done, err := nfdh.handlePrune(ctx, &nfdCR)
		Expect(err).To(BeNil())
		Expect(done).To(BeTrue())
	})

	It("failed to get prune job from the cluster", func() {
		nfdCR.Spec.PruneOnDelete = true
		mockJob.EXPECT().GetJob(ctx, namespace, "nfd-prune").Return(nil, fmt.Errorf("some error"))

		done, err := nfdh.handlePrune(ctx, &nfdCR)

		Expect(err).To(HaveOccurred())
		Expect(done).To(BeFalse())
	})

	It("job does not exists, creating it fails", func() {
		nfdCR.Spec.PruneOnDelete = true
		gomock.InOrder(
			mockJob.EXPECT().GetJob(ctx, namespace, "nfd-prune").Return(nil, apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockJob.EXPECT().CreatePruneJob(ctx, &nfdCR).Return(fmt.Errorf("some error")),
		)

		done, err := nfdh.handlePrune(ctx, &nfdCR)

		Expect(err).To(HaveOccurred())
		Expect(done).To(BeFalse())
	})

	It("job does not exists, creating it succeeds", func() {
		nfdCR.Spec.PruneOnDelete = true
		gomock.InOrder(
			mockJob.EXPECT().GetJob(ctx, namespace, "nfd-prune").Return(nil, apierrors.NewNotFound(schema.GroupResource{}, "whatever")),
			mockJob.EXPECT().CreatePruneJob(ctx, &nfdCR).Return(nil),
		)

		done, err := nfdh.handlePrune(ctx, &nfdCR)

		Expect(err).To(BeNil())
		Expect(done).To(BeFalse())
	})

	DescribeTable("prune job exsists flows", func(podFailed, podSucceeded bool) {
		nfdCR.Spec.PruneOnDelete = true
		foundJob := batchv1.Job{}
		if podFailed {
			foundJob.Status.Failed = 1
		}
		if podSucceeded {
			foundJob.Status.Succeeded = 1
		}
		mockJob.EXPECT().GetJob(ctx, namespace, "nfd-prune").Return(&foundJob, nil)

		done, err := nfdh.handlePrune(ctx, &nfdCR)

		switch {
		case !podFailed && !podSucceeded:
			Expect(err).To(BeNil())
			Expect(done).To(BeFalse())
		case podFailed:
			Expect(err).To(HaveOccurred())
			Expect(done).To(BeFalse())
		case podSucceeded:
			Expect(err).To(BeNil())
			Expect(done).To(BeTrue())
		}
	},
		Entry("job has not finished yet", false, false),
		Entry("job finished, its pod successfull", false, true),
		Entry("job finished, its pod failed", true, false),
	)
})

var _ = Describe("handleStatus", func() {
	var (
		ctrl       *gomock.Controller
		clnt       *client.MockClient
		mockStatus *status.MockStatusAPI
		nfdh       nodeFeatureDiscoveryHelperAPI
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		clnt = client.NewMockClient(ctrl)
		mockStatus = status.NewMockStatusAPI(ctrl)
		nfdh = newNodeFeatureDiscoveryHelperAPI(clnt, nil, nil, nil, nil, nil, mockStatus, scheme)
	})

	ctx := context.Background()
	nfdCR := nfdv1.NodeFeatureDiscovery{
		Status: nfdv1.NodeFeatureDiscoveryStatus{
			Conditions: []metav1.Condition{},
		},
	}
	newConditions := []metav1.Condition{}

	It("conditions are equal, no status update is needed", func() {
		gomock.InOrder(
			mockStatus.EXPECT().GetConditions(ctx, &nfdCR).Return(newConditions),
			mockStatus.EXPECT().AreConditionsEqual(newConditions, nfdCR.Status.Conditions).Return(true),
		)

		err := nfdh.handleStatus(ctx, &nfdCR)
		Expect(err).To(BeNil())
	})

	It("conditions are not equal, status update is needed", func() {
		statusWriter := client.NewMockStatusWriter(ctrl)
		expectedNFD := nfdv1.NodeFeatureDiscovery{
			Status: nfdv1.NodeFeatureDiscoveryStatus{
				Conditions: newConditions,
			},
		}
		gomock.InOrder(
			mockStatus.EXPECT().GetConditions(ctx, &nfdCR).Return(newConditions),
			mockStatus.EXPECT().AreConditionsEqual(newConditions, nfdCR.Status.Conditions).Return(false),
			clnt.EXPECT().Status().Return(statusWriter),
			statusWriter.EXPECT().Patch(ctx, &expectedNFD, gomock.Any()).Return(nil),
		)

		err := nfdh.handleStatus(ctx, &nfdCR)
		Expect(err).To(BeNil())
	})

	It("conditions are not equal, status update failed", func() {
		statusWriter := client.NewMockStatusWriter(ctrl)
		expectedNFD := nfdv1.NodeFeatureDiscovery{
			Status: nfdv1.NodeFeatureDiscoveryStatus{
				Conditions: newConditions,
			},
		}
		gomock.InOrder(
			mockStatus.EXPECT().GetConditions(ctx, &nfdCR).Return(newConditions),
			mockStatus.EXPECT().AreConditionsEqual(newConditions, nfdCR.Status.Conditions).Return(false),
			clnt.EXPECT().Status().Return(statusWriter),
			statusWriter.EXPECT().Patch(ctx, &expectedNFD, gomock.Any()).Return(fmt.Errorf("some error")),
		)

		err := nfdh.handleStatus(ctx, &nfdCR)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("getOperandImage", Ordered, func() {
	var nfdCR nfdv1.NodeFeatureDiscovery
	const (
		testImage      = "TestRegistry/TestNamespace/TestImage:latest"
		otherTestImage = "TestRegistry/TestNamespace/SomeOtherTestImage:latest"
	)
	BeforeEach(func() {
		nfdCR = nfdv1.NodeFeatureDiscovery{}
	})
	BeforeAll(func() {
		err := os.Setenv("NODE_FEATURE_DISCOVERY_IMAGE", testImage)
		Expect(err).ToNot(HaveOccurred())
	})
	AfterAll(func() {
		err := os.Unsetenv("NODE_FEATURE_DISCOVERY_IMAGE")
		Expect(err).ToNot(HaveOccurred())
	})
	It("Should get the operand image from the env variable", func() {
		operandImage := getOperandImage(&nfdCR)
		Expect(operandImage).To(Equal(testImage))
	})
	It("Should get the operand image from the cr", func() {
		nfdCR.Spec.Operand.Image = otherTestImage
		operandImage := getOperandImage(&nfdCR)
		Expect(operandImage).To(Equal(otherTestImage))
	})
})
