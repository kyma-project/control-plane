FROM golang:1.14-alpine as builder

ARG DOCK_PKG_DIR=/go/src/github.com/kyma-project/control-plane/tools/provisioner-tools/provider-config-migration-job

WORKDIR $DOCK_PKG_DIR
COPY . $DOCK_PKG_DIR

COPY ./licenses ${DOCK_PKG_DIR}/licenses

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o providerconfigmigrationjob ./cmd/

FROM scratch
LABEL source=git@github.com:kyma-project/kyma.git

COPY --from=builder /go/src/github.com/kyma-project/control-plane/tools/provisioner-tools/provider-config-migration-job/providerconfigmigrationjob .
COPY --from=builder /go/src/github.com/kyma-project/control-plane/tools/provisioner-tools/provider-config-migration-job/licenses ./licenses

CMD ["/providerconfigmigrationjob"]