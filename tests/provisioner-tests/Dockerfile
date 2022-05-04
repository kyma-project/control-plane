FROM golang:1.14-alpine3.12 as builder

ENV SRC_DIR=/go/src/github.com/kyma-project/control-plane/tests/provisioner-tests

WORKDIR $SRC_DIR
COPY . $SRC_DIR

COPY go.mod .
COPY go.sum .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go test -c ./test/provisioner



FROM alpine:3.12
LABEL source=git@github.com:kyma-project/kyma.git

WORKDIR /app

RUN apk --no-cache add ca-certificates curl

COPY --from=builder /go/src/github.com/kyma-project/control-plane/tests/provisioner-tests/scripts/entrypoint.sh .
COPY --from=builder /go/src/github.com/kyma-project/control-plane/tests/provisioner-tests/provisioner.test .
COPY --from=builder /go/src/github.com/kyma-project/control-plane/tests/provisioner-tests/licenses ./licenses

ENTRYPOINT ["./entrypoint.sh"]
