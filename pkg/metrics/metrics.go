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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// When adding metric names, see https://prometheus.io/docs/practices/naming/#metric-names
const (
	degradedInfoQuery = "nfd_degraded_info"
	buildInfoQuery    = "nfd_build_info"
	instanceInfoQuery = "nfd_instance_info"
)

var (
	version = "undefined"
	instanceInfo = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: instanceInfoQuery,
			Help: "A metric with a constant '1' value labeled instance from which NFD is storing annotations",
		},
		[]string{"instance", "namespace"},
	)
	degradedState = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: degradedInfoQuery,
			Help: "Indicates whether the Node Feature Discovery Operator is degraded.",
		},
	)
	buildInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: buildInfoQuery,
			Help: "A metric with a constant '1' value labeled version from which Node Feature Discovery Operator was built.",
		},
		[]string{"version"},
	)

	instanceList = make(map[string]bool)
)

// registerVersion exposes the Operator build version.
func registerVersion(version string) {
	buildInfo.WithLabelValues(version).Set(1)
}

// RegisterInstance sets the metric that registers if NFD running instances
func RegisterInstance(instance string, namespace string) {

	if !instanceList[instance] {
		instanceList[instance] = true
		instanceInfo.WithLabelValues(instance, namespace).Inc()
	}
}

// Degraded sets the metric that indicates whether the operator is in degraded
// mode or not.
func Degraded(deg bool) {
	if deg {
		degradedState.Set(1)
		return
	}
	degradedState.Set(0)
}

// Register custom metrics with the global prometheus registry
func init() {
	metrics.Registry.MustRegister(
		degradedState,
		buildInfo,
		instanceInfo,
	)

	registerVersion(version)
}
