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

package config

import (
	"os"
	"strconv"
	"time"
)

const (
	nodeFeatureDiscoveryImageDefault string = "quay.io/openshift/origin-node-feature-discovery:4.9"
	contextTimeout                          = 300 * time.Second
	// A number in seconds to define a context Timeout
	// E.g. if 5 seconds is wanted, the CTX_TIMEOUT=5
	contextTimeoutEnvVar = "CTX_TIMEOUT"
)

// Config hosts different parameters that
type Config struct {
	CtxTimeOut time.Duration
}

// ManagerOptions contains configurable options for the Shipwright build controller manager
type ManagerOptions struct {
	LeaderElectionNamespace string
	LeaseDuration           *time.Duration
	RenewDeadline           *time.Duration
	RetryPeriod             *time.Duration
}

// NewDefaultConfig returns a new Config, with context timeout and default Kaniko image.
func NewDefaultConfig() *Config {
	return &Config{
		CtxTimeOut: contextTimeout,
	}
}

// SetConfigFromEnv updates the configuration managed by environment variables.
func (c *Config) SetConfigFromEnv() error {
	if timeout := os.Getenv(contextTimeoutEnvVar); timeout != "" {
		i, err := strconv.Atoi(timeout)
		if err != nil {
			return err
		}
		c.CtxTimeOut = time.Duration(i) * time.Second
	}

	return nil
}

// NodeFeatureDiscoveryImage returns the operator's operand/nfd image.
func NodeFeatureDiscoveryImage() string {
	nodeFeatureDiscoveryImage := os.Getenv("NODE_FEATURE_DISCOVERY_IMAGE")

	if len(nodeFeatureDiscoveryImage) > 0 {
		return nodeFeatureDiscoveryImage
	}

	return nodeFeatureDiscoveryImageDefault
}
