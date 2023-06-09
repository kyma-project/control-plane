FROM europe-docker.pkg.dev/kyma-project/prod/external/golang:1.20.3-alpine3.17 as builder

ENV SRC_DIR=/go/src/github.com/kyma-project/control-plane/tests/provisioner-tests

WORKDIR $SRC_DIR
COPY . $SRC_DIR

COPY go.mod .
COPY go.sum .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go test -c ./test/provisioner


FROM europe-docker.pkg.dev/kyma-project/prod/external/alpine:3.17.3
LABEL source=git@github.com:kyma-project/kyma.git

WORKDIR /app

RUN apk --no-cache add ca-certificates curl

COPY --from=builder /go/src/github.com/kyma-project/control-plane/tests/provisioner-tests/scripts/entrypoint.sh .
COPY --from=builder /go/src/github.com/kyma-project/control-plane/tests/provisioner-tests/provisioner.test .

ENTRYPOINT ["./entrypoint.sh"]
