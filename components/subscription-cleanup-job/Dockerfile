FROM golang:1.14-alpine as builder

ARG DOCK_PKG_DIR=/go/src/github.com/kyma-project/control-plane/components/subscription-cleanup-job

WORKDIR $DOCK_PKG_DIR
COPY . $DOCK_PKG_DIR

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o subscriptioncleanupjob ./cmd/

FROM scratch
LABEL source=git@github.com:kyma-project/kyma.git

COPY --from=builder /go/src/github.com/kyma-project/control-plane/components/subscription-cleanup-job/subscriptioncleanupjob .
COPY --from=builder /go/src/github.com/kyma-project/control-plane/components/subscription-cleanup-job/licenses ./licenses

CMD ["/subscriptioncleanupjob"]