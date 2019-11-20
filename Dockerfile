FROM registry.access.redhat.com/ubi8/go-toolset AS builder
WORKDIR /go/src/github.com/openshift/cluster-nfd-operator
COPY . .
RUN make build

FROM registry.access.redhat.com/ubi8/ubi
ARG CSV=4.2
COPY --from=builder /go/src/github.com/openshift/cluster-nfd-operator/cluster-nfd-operator /usr/bin/

RUN mkdir -p /etc/kubernetes/node-feature-discovery/assets
COPY assets/ /etc/kubernetes/node-feature-discovery/assets

#ADD controller-manifests /manifests
COPY deploy/olm-catalog/$CSV /manifests/$CSV
COPY deploy/olm-catalog/nfd.package.yaml /manifests/

RUN useradd cluster-nfd-operator
USER cluster-nfd-operator
ENTRYPOINT ["/usr/bin/cluster-nfd-operator"]
LABEL io.k8s.display-name="OpenShift cluster-nfd-operator" \
      io.k8s.description="This is a component of OpenShift and manages the node feature discovery." \
      io.openshift.tags="openshift" \
      com.redhat.delivery.appregistry=true \
      maintainer="Performance Sensitive Application Platform (PSAP) <openshift-psap@redhat.com>"


