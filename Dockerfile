# Build the manager binary
FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.16-openshift-4.9 as builder
WORKDIR /go/src/github.com/openshift/cluster-nfd-operator

# Build
COPY . .
RUN make build

# Create production image for running the operator
FROM registry.ci.openshift.org/ocp/4.9:base
ARG CSV=4.10
COPY --from=builder /go/src/github.com/openshift/cluster-nfd-operator/node-feature-discovery-operator /

RUN mkdir -p /opt/nfd
COPY build/assets /opt/nfd
COPY manifests /manifests

RUN useradd cluster-nfd-operator
USER cluster-nfd-operator

ENTRYPOINT ["/node-feature-discovery-operator"]
LABEL io.k8s.display-name="node-feature-discovery-operator"