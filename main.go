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
	"flag"
	"fmt"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog/v2"

	securityscheme "github.com/openshift/client-go/security/clientset/versioned/scheme"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	nfdopenshiftv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"github.com/openshift/cluster-nfd-operator/controllers"
	"github.com/openshift/cluster-nfd-operator/pkg/config"
	"github.com/openshift/cluster-nfd-operator/pkg/version"
	// +kubebuilder:scaffold:imports
)

var (
	// scheme holds a new scheme for the operator
	scheme = runtime.NewScheme()
)

const (
	// ProgramName is the canonical name of this program
	ProgramName = "nfd-operator"
)

// operatorArgs holds command line arguments
type operatorArgs struct {
	metricsAddr          string
	enableLeaderElection bool
	probeAddr            string
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(securityscheme.AddToScheme(scheme))

	utilruntime.Must(nfdopenshiftv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	// probeAddr is responsible for the health probe bind address, where the health
	// probe is responsible for determining liveness, readiness, and configuration
	// of the operator pods. Note that the port which is being binded must match
	// the bind port under './config' and './manifests'
	probeAddr := ":8081"

	flags := flag.NewFlagSet(ProgramName, flag.ExitOnError)

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

	watchNamespace, err := config.GetWatchNamespace()
	if err != nil {
		klog.Info("unable to get WatchNamespace, " +
			"the manager will watch and manage resources in all namespaces")
	}

	// Create a new manager to manage the operator
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     args.metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         args.enableLeaderElection,
		LeaderElectionID:       "39f5e5c3.nodefeaturediscoveries.nfd.openshift.io",
		Namespace:              watchNamespace,
	})

	if err != nil {
		klog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.NodeFeatureDiscoveryReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		klog.Error(err, "unable to create controller", "controller", "NodeFeatureDiscovery")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	// Next, add a Healthz checker to the manager. Healthz is a health and liveness package
	// that the operator will use to periodically check the health of its pods, etc.
	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	// Now add a ReadyZ checker to the manager as well. It is important to ensure that the
	// API server's readiness is checked when the operator is installed and running.
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Register signal handler for SIGINT and SIGTERM to terminate the manager
	klog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		klog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func initFlags(flagset *flag.FlagSet) *operatorArgs {
	args := operatorArgs{}

	// Setup CLI arguments
	flagset.StringVar(&args.metricsAddr, "metrics-bind-address", ":8080", "The address the Prometheus "+
		"metric endpoint binds to for scraping NFD resource usage data.")
	flagset.StringVar(&args.probeAddr, "health-probe-bind-address", ":8081", "The address the probe "+
		"endpoint binds to for determining liveness, readiness, and configuration of"+
		"operator pods.")
	flagset.BoolVar(&args.enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	return &args
}
