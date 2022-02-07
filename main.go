/*

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

package main

import (
	"context"
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog/v2"

	securityscheme "github.com/openshift/client-go/security/clientset/versioned/scheme"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nfdkubernetesiov1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"github.com/openshift/cluster-nfd-operator/controllers"
	"github.com/openshift/cluster-nfd-operator/pkg/utils"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(securityscheme.AddToScheme(scheme))

	utilruntime.Must(nfdopenshiftv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func printVersion() {
	klog.Infof("Operator Version: %s", version.Version)
}

	printVersion := flags.Bool("version", false, "Print version and exit.")

	args := initFlags(flags)
	// Inject klog flags
	klog.InitFlags(flags)

	_ = flags.Parse(os.Args[1:])
	if len(flags.Args()) > 0 {
		fmt.Fprintf(flags.Output(), "unknown command line argument: %s\n", flags.Args()[0])
		flags.Usage()
		os.Exit(2)
	}

	if *printVersion {
		fmt.Println(ProgramName, version.Get())
		os.Exit(0)
	}

<<<<<<< HEAD
	watchNamespace, envSet := utils.GetWatchNamespace()
	if !envSet {
		klog.Info("unable to get WatchNamespace, " +
			"the manager will watch and manage resources in all namespaces")
	}
=======
	watchNamespace, err := utils.GetWatchNamespace()
	if err != nil {
		klog.Error(err, "unable to get WatchNamespace, "+
			"the manager will watch and manage resources in all namespaces")
	}

	// Create a new manager to manage the operator
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     args.metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: args.probeAddr,
		LeaderElection:         args.enableLeaderElection,
		LeaderElectionID:       "39f5e5c3.nodefeaturediscoveries.nfd.kubernetes.io",
		Namespace:              watchNamespace,
	})
>>>>>>> 7c09709f (Drop operand.namespace from CRD)

	if err := labelNamespace(watchNamespace); err != nil {
		setupLog.V(2).Error(err, "unable to update Namespace, "+watchNamespace+
			" the manager won't expose metrics and alerts")
	}

	restConfig := ctrl.GetConfigOrDie()
	le := leaderelection.GetLeaderElectionConfig(restConfig, enableLeaderElection)

	mgr, err := ctrl.NewManager(restConfig, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaseDuration:          &le.LeaseDuration.Duration,
		RenewDeadline:          &le.RenewDeadline.Duration,
		RetryPeriod:            &le.RetryPeriod.Duration,
		LeaderElectionID:       "nfd.openshift.io",
		Namespace:              watchNamespace, // namespaced-scope when the value is not an empty string
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.NodeFeatureDiscoveryReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NodeFeatureDiscovery")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
