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

package leaderelection

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	openshiftcorev1 "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const infraResourceName = "cluster"

// GetLeaderElectionConfig returns leader election configs defaults based on the cluster topology
func GetLeaderElectionConfig(restConfig *rest.Config, enabled bool) configv1.LeaderElection {

	// Defaults follow conventions
	// https://github.com/openshift/enhancements/blob/master/CONVENTIONS.md#high-availability
	defaultLeaderElection := leaderElectionDefaulting(
		configv1.LeaderElection{
			Disable: !enabled,
		},
		"", "",
	)

	if enabled {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		if infra, err := getClusterInfraStatus(ctx, restConfig); err == nil && infra != nil {
			if infra.ControlPlaneTopology == configv1.SingleReplicaTopologyMode {
				return leaderElectionSNOConfig(defaultLeaderElection)
			}
		} else {
			klog.Warningf("unable to get cluster infrastructure status, using HA cluster values for leader election: %v", err)
		}
	}

	return defaultLeaderElection
}

func leaderElectionDefaulting(config configv1.LeaderElection, defaultNamespace, defaultName string) configv1.LeaderElection {
	ret := *(&config).DeepCopy()

	// We want to be able to tolerate 60s of kube-apiserver disruption without causing pod restarts.
	// We want the graceful lease re-acquisition fairly quick to avoid waits on new deployments and other rollouts.
	// We want a single set of guidance for nearly every lease in openshift.  If you're special, we'll let you know.
	// 1. clock skew tolerance is leaseDuration-renewDeadline == 30s
	// 2. kube-apiserver downtime tolerance is == 78s
	//      lastRetry=floor(renewDeadline/retryPeriod)*retryPeriod == 104
	//      downtimeTolerance = lastRetry-retryPeriod == 78s
	// 3. worst non-graceful lease acquisition is leaseDuration+retryPeriod == 163s
	// 4. worst graceful lease acquisition is retryPeriod == 26s
	if ret.LeaseDuration.Duration == 0 {
		ret.LeaseDuration.Duration = 137 * time.Second
	}

	if ret.RenewDeadline.Duration == 0 {
		// this gives 107/26=4 retries and allows for 137-107=30 seconds of clock skew
		// if the kube-apiserver is unavailable for 60s starting just before t=26 (the first renew),
		// then we will retry on 26s intervals until t=104 (kube-apiserver came back up at 86), and there will
		// be 33 seconds of extra time before the lease is lost.
		ret.RenewDeadline.Duration = 107 * time.Second
	}
	if ret.RetryPeriod.Duration == 0 {
		ret.RetryPeriod.Duration = 26 * time.Second
	}
	if len(ret.Namespace) == 0 {
		if len(defaultNamespace) > 0 {
			ret.Namespace = defaultNamespace
		} else {
			// Fall back to the namespace associated with the service account token, if available
			if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
				if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
					ret.Namespace = ns
				}
			}
		}
	}
	if len(ret.Name) == 0 {
		ret.Name = defaultName
	}
	return ret
}

func leaderElectionSNOConfig(config configv1.LeaderElection) configv1.LeaderElection {
	// We want to make sure we respect a 30s clock skew as well as a 4 retry attempt with out making
	// leader election ineffectual while still having some small performance gain by limiting calls against
	// the api server.

	// 1. clock skew tolerance is leaseDuration-renewDeadline == 30s
	// 2. kube-apiserver downtime tolerance is == 180s
	//      lastRetry=floor(renewDeadline/retryPeriod)*retryPeriod == 240
	//      downtimeTolerance = lastRetry-retryPeriod == 180s
	// 3. worst non-graceful lease acquisition is leaseDuration+retryPeriod == 330s
	// 4. worst graceful lease acquisition is retryPeriod == 60s

	ret := *(&config).DeepCopy()
	// 270-240 = 30s of clock skew tolerance
	ret.LeaseDuration.Duration = 270 * time.Second
	// 240/60 = 4 retries attempts before leader is lost.
	ret.RenewDeadline.Duration = 240 * time.Second
	// With 60s retry config we aim to maintain 30s of clock skew as well as 4 retry attempts.
	ret.RetryPeriod.Duration = 60 * time.Second
	return ret
}

func getClusterInfraStatus(ctx context.Context, restClient *rest.Config) (*configv1.InfrastructureStatus, error) {
	client, err := openshiftcorev1.NewForConfig(restClient)
	if err != nil {
		return nil, err
	}
	infra, err := client.Infrastructures().Get(ctx, infraResourceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if infra == nil {
		return nil, fmt.Errorf("getting resource Infrastructure (name: %s) succeeded but object was nil", infraResourceName)
	}
	return &infra.Status, nil
}
