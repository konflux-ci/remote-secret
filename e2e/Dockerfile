ARG TARGETARCH=arm64
ARG GO_VERSION=1.19
FROM golang:$GO_VERSION-buster as builder
#FROM registry.access.redhat.com/ubi9/ubi-minimal:9.2-750.1697625013 as builder
#FROM registry.access.redhat.com/ubi9/go-toolset:1.19.13 as builder

ENV KUBECTL_VERSION="v1.24.3"
ENV HELM_VERSION="v3.7.1"

RUN go install github.com/onsi/ginkgo/v2/ginkgo@v2.1.6
RUN wget -q https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/arm64/kubectl -O /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    wget -q https://get.helm.sh/helm-${HELM_VERSION}-linux-arm64.tar.gz -O - | tar -xzO linux-arm64/helm > /usr/local/bin/helm && \
    chmod +x /usr/local/bin/helm

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.2-750.1697625013

COPY --from=builder /go/bin/ginkgo /usr/local/bin/
COPY --from=builder /usr/local/bin/kubectl /usr/local/bin/
COPY --from=builder /usr/local/bin/helm /usr/local/bin/

COPY entrypoint.sh                   /entrypoint.sh
COPY suites/provider/provider.test   /provider.test
COPY k8s                             /k8s

CMD [ "/entrypoint.sh" ]
