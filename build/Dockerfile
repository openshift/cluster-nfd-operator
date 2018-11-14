FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ADD build/_output/bin/cluster-nfd-operator /usr/local/bin/cluster-nfd-operator
