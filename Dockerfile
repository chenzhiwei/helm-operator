# Build the manager binary
FROM docker.io/library/golang:1.17 as builder

WORKDIR /workspace
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
COPY utils/ utils/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
ARG VCS_URL=https://github.com/chenzhiwei/helm-operator
ARG VCS_REF=master
ARG BUILD_DATE
LABEL org.label-schema.vcs-url=$VCS_URL \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.build-date=$BUILD_DATE \
      org.label-schema.schema-version="1.0"

WORKDIR /
COPY --from=builder /workspace/manager .
COPY config/crd/bases/app.siji.io_helmcharts.yaml ./config/crd/bases/app.siji.io_helmcharts.yaml
COPY config/crd/bases/app.siji.io_helmdogs.yaml ./config/crd/bases/app.siji.io_helmdogs.yaml
COPY config/webhook/manifests.yaml ./config/webhook/manifests.yaml
COPY config/webhook/service.yaml ./config/webhook/service.yaml
USER 65532:65532

ENTRYPOINT ["/manager"]
