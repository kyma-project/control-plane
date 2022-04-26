FROM eu.gcr.io/kyma-project/test-infra/buildpack-golang:v20211116-11e89837 as builder


ENV BASE_APP_DIR /go/src/github.com/kyma-project/control-plane/components/kubeconfig-service
ENV CGO_ENABLED 0
WORKDIR ${BASE_APP_DIR}
#
# Copy files
#

COPY . .

#
# Build app
#
RUN go build -v -o main ./cmd/generator/main.go
RUN mkdir /app && mv ./main /app/main && mv ./licenses /app/licenses

LABEL source = git@github.com:kyma-project/control-plane.git

WORKDIR /app

#
# Install certificates
#
FROM alpine:3.15.4
RUN  apk add --no-cache ca-certificates openssl
#
# Copy binary
#

COPY --from=builder /app /app

#
# Run app
#

CMD ["/app/main"]
