# Node Feature Discovery Operator
 The Node Feature Discovery operator is a tool for Openshift administrators that makes it easy to detect and understand the hardware features and 
 configurations of a cluster's nodes. With this operator, administrators can easily gather information about their nodes that can be used for scheduling,
 resource management, and more by controlling the life cycle of [NFD](https://github.com/kubernetes-sigs/node-feature-discovery).

## Upstream Project

[Node Feature Discovery](https://github.com/kubernetes-sigs/node-feature-discovery) – a Kubernetes add-on for detecting hardware features and system configuration.

The [Node Feature Discovery](https://kubernetes-sigs.github.io/node-feature-discovery/stable/get-started/index.html) and [Node Feature Discovery operator](https://kubernetes-sigs.github.io/node-feature-discovery-operator/stable/introduction/index.html) are Upstream projects under the [kubernetes-Sigs](https://github.com/kubernetes-sigs) organization

## Getting started with the Node Feature Discovery Operator

> Prerequisite: a running OpenShift cluster 4.6+

Get the source code

```bash
git clone https://github.com/openshift/cluster-nfd-operator
```

Deploy the operator

```bash
IMAGE_REGISTRY=quay.io/<your-personal-registry>
make image push deploy
```

Create a NodeFeatureDiscovery instance

```bash
oc apply -f config/samples/nfd.openshift.io_v1_nodefeaturediscovery.yaml
```

## Verify

The Operator will deploy NFD based on the information
on the NodeFeatureDiscovery CR instance,
after a moment you should be able to see

```bash
$ oc -n openshift-nfd get ds,deploy
NAME                        DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/nfd-worker   3         3         3       3            3           <none>          5s
NAME                         READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/nfd-master   1/1     1            1           17s
```

Check that NFD feature labels have been created

```bash
$ oc get node -o json | jq .items[].metadata.labels
{
  "beta.kubernetes.io/arch": "amd64",
  "beta.kubernetes.io/os": "linux",
  "feature.node.kubernetes.io/cpu-cpuid.ADX": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AESNI": "true",
  "feature.node.kubernetes.io/cpu-cpuid.AVX": "true",
...
```

## Extending NFD with sidecar containers and hooks

First see upstream documentation of the hook feature and how to create a correct hook file:
https://github.com/kubernetes-sigs/node-feature-discovery#local-user-specific-features.

The DaemonSet running on the workers will mount the `hostPath: /etc/kubernetes/node-feature-discovery/source.d`. Additional hooks can than be provided by a sidecar container that is as well running on the workers and mounting the same hostpath and writing the hook executable (shell-script, compiled code, ...) to this directory.

NFD will execute any file in this directory, if one needs any configuration for the hook, a separate configuration directory can be created under `/etc/kubernetes/node-feature-discovery/source.d` e.g. `/etc/kubernetes/node-feature-discovery/source.d/own-hook-conf`, NFD will not recurse deeper into the file hierarchy.

## Building NFD operator for ARM locally

There 2 options:

1) Using ARM server - process is the same, but you should use Dockerfile.arm instead of Dockerfile
2) Using x86 server/laptop - process is the same but before running build command, Makefile.arm should be copied into Makefile
