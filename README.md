# Node Feature Discovery Operator
 The Cluster Node Feature Discovery operator manages detection of hardware features and configuration in a Openshift cluster. The operator orchestrates all resources needed to run the NFD DaemonSet (Upstream: https://github.com/kubernetes-sigs/node-feature-discovery)
 
## Building the operator
Checkout the sources
```
$ git clone https://github.com/openshift/cluster-nfd-operator $GOPATH/src/github.com/openshift/cluster-nfd-operator
```

Update the `Makefile` and edit `IMAGE_TAG` and `IMAGE_REGISTRY` one will need those later to update the operator manifest (`image: $IMAGE_REGISTRY/$IMAGE_TAG`).

Update the Dockerfile with the correct golang build image and the base image the operator will run with. 

## Manual deploy of the operator
Checkout the sources
```
$ git clone https://github.com/openshift/cluster-nfd-operator
```
Update the  `Makefile` with the a custom image built and configure the namespace where the operator should be deployed.

The default CR will create the operand (NFD) in the `openshift-nfd` namespace, the CR can be edited to choose another namespace and image. See the `manifests/0700_cr.yaml` for the default values.

```
$ cd cluster-nfd-operator/manifests
$ make deploy
```
The operator will use the NFD image built from: https://github.com/openshift/node-feature-discovery

To uninstall the operator run 
```
$ make undeploy
```

To verify the correct working of NFD a e2e test can be run as well: 
```
$ make test-e2e
```

## Extending NFD with sidecar containers and hooks

First see upstream documentation of the hook feature and how to create a correct hook file: 
https://github.com/kubernetes-sigs/node-feature-discovery#local-user-specific-features.

The DaemonSet running on the workers will mount the `hostPath: /etc/kubernetes/node-feature-discovery/source.d`. Additional hooks can than be provided by a sidecar container that is as well running on the workers and mounting the same hostpath and writing the hook executable (shell-script, compiled code, ...) to this directory. 

NFD will execute any file in this directory, if one needs any configuration for the hook, a separate configuration directory can be created under `/etc/kubernetes/node-feature-discovery/source.d` e.g. `/etc/kubernetes/node-feature-discovery/source.d/own-hook-conf`, NFD will not recurse deeper into the file hierarchy. 


