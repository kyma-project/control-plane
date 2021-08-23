FROM eu.gcr.io/kyma-project/external/golang:1.16.3-alpine as builder

ENV BASE_APP_DIR /go/src/github.com/kyma-project/control-plane/components/provisioner
WORKDIR ${BASE_APP_DIR}

ENV GO111MODULES=on

COPY ./go.mod ${BASE_APP_DIR}/go.mod
COPY ./go.sum ${BASE_APP_DIR}/go.sum

# Run go mod download first to take advantage of Docker caching
RUN apk add git && go mod download

#COPY ./licenses ${BASE_APP_DIR}/licenses
COPY ./cmd/ ${BASE_APP_DIR}/cmd
COPY ./third_party/ ${BASE_APP_DIR}/third_party
COPY ./pkg/ ${BASE_APP_DIR}/pkg/
COPY ./internal/ ${BASE_APP_DIR}/internal/

RUN apk add -U --no-cache ca-certificates && update-ca-certificates

RUN go build -v -o main ./cmd/
RUN mkdir /app && mv ./main /app/main
#RUN mv ./licenses /app/licenses

FROM eu.gcr.io/kyma-project/external/alpine:3.13.5
LABEL source = git@github.com:kyma-project/control-plane.git

WORKDIR /app

COPY --from=builder /app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/app/main"]
