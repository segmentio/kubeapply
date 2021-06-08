# Fetch or build all required binaries
FROM golang:1.16 as builder

ARG VERSION_REF
RUN test -n "${VERSION_REF}"

ENV SRC github.com/segmentio/kubeapply

RUN apt-get update && apt-get install --yes \
    curl \
    wget

COPY . /go/src/${SRC}
RUN cd /usr/local/bin && /go/src/${SRC}/scripts/pull-deps.sh

WORKDIR /go/src/${SRC}

ENV CGO_ENABLED=0
ENV GO111MODULE=on

RUN make kubeapply VERSION_REF=${VERSION_REF} && \
    cp build/kubeapply /usr/local/bin
RUN make terraform-provider-kubeapply VERSION_REF=${VERSION_REF} && \
    cp build/terraform-provider-kubeapply /usr/local/bin

# Copy into final image
FROM ubuntu:18.04

RUN apt-get update && apt-get install --yes \
    curl \
    git \
    python3 \
    python3-pip

RUN pip3 install awscli

COPY --from=builder \
    /usr/local/bin/helm \
    /usr/local/bin/kubectl \
    /usr/local/bin/kubeapply \
    /usr/local/bin/terraform-provider-kubeapply \
    /usr/local/bin/
