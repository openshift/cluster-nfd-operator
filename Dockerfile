FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.15-openshift-4.7 AS builder
WORKDIR /go/src/github.com/openshift/cluster-nfd-operator
COPY . .
RUN make build

FROM registry.ci.openshift.org/ocp/4.7:base
ARG CSV=4.7
COPY --from=builder /go/src/github.com/openshift/cluster-nfd-operator/cluster-nfd-operator /usr/bin/

RUN mkdir -p /opt/nfd
COPY assets /opt/nfd

#ADD controller-manifests /manifests
COPY manifests/olm-catalog/$CSV /manifests/$CSV
COPY manifests/olm-catalog/nfd.package.yaml /manifests/

RUN useradd cluster-nfd-operator
USER cluster-nfd-operator
ENTRYPOINT ["/usr/bin/cluster-nfd-operator"]
LABEL io.k8s.display-name="OpenShift cluster-nfd-operator" \
      io.k8s.description="This is a component of OpenShift and manages the node feature discovery." \
      io.openshift.tags="openshift" \
      com.redhat.delivery.appregistry=False \
      maintainer="ATS Auto Tuning Scalability  <aos-scalability@redhat.com>"


