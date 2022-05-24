# Build the manager binary
FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.18-openshift-4.11 AS builder
WORKDIR /go/src/github.com/openshift/cluster-nfd-operator

# Build
COPY . .
RUN make build

# Create production image for running the operator
FROM registry.ci.openshift.org/ocp/4.11:base

ARG CSV=4.11
COPY --from=builder /go/src/github.com/openshift/cluster-nfd-operator/node-feature-discovery-operator /

RUN mkdir -p /opt/nfd
COPY build/assets /opt/nfd
COPY manifests /manifests

RUN useradd cluster-nfd-operator
USER cluster-nfd-operator

ENTRYPOINT ["/node-feature-discovery-operator"]
LABEL io.k8s.display-name="node-feature-discovery-operator" \
      io.k8s.description="This is the image for the Node Feature Discovery Operator."
