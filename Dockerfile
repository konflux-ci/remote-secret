# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset:1.20.10 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /opt/app-root/src
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.3-1361 as remote-secret-operator
# Install the 'shadow-utils' which contains `adduser` and `groupadd` binaries
RUN microdnf update -y \
    && microdnf -y --setopt=tsflags=nodocs install shadow-utils \
    && microdnf -y reinstall tzdata \
	&& groupadd --gid 65532 nonroot \
	&& adduser \
		--no-create-home \
		--no-user-group \
		--uid 65532 \
		--gid 65532 \
		nonroot \
    && microdnf -y clean all \
    && rm -rf /var/cache/yum
WORKDIR /
COPY --from=builder /opt/app-root/src/manager .
# It is mandatory to set these labels
LABEL description="RHTAP RemoteSecret Operator"
LABEL io.k8s.description="RHTAP RemoteSecret Operator"
LABEL io.k8s.display-name="remotesecret-operator"
LABEL summary="RHTAP RemoteSecret Operator"
LABEL io.openshift.tags="rhtap"
USER 65532:65532

ENTRYPOINT ["/manager"]
