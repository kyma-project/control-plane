FROM golang:1.22.0-alpine3.19 as builder

ENV SRC_DIR=/go/src/github.com/kyma-project/control-plane/tests/e2e/provisioning

WORKDIR $SRC_DIR
COPY . $SRC_DIR

RUN CGO_ENABLED=0 GOOS=linux go test -ldflags="-s -w" -c ./test

FROM europe-docker.pkg.dev/kyma-project/prod/external/alpine:3.19.1
LABEL source=git@github.com:kyma-project/control-plane.git

WORKDIR /app

RUN apk --no-cache add ca-certificates curl

COPY --from=builder /go/src/github.com/kyma-project/control-plane/tests/e2e/provisioning/test.test .

ENTRYPOINT ["/app/test.test"]
