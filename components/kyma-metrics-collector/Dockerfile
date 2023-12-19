FROM europe-docker.pkg.dev/kyma-project/prod/external/golang:1.21.5-alpine3.19 as builder

ENV BASE_APP_DIR /go/src/github.com/kyma-project/control-plane/components/kyma-metrics-collector
WORKDIR ${BASE_APP_DIR}

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o kyma-metrics-collector ./cmd/main.go
RUN mkdir /app && mv ./kyma-metrics-collector /app/kyma-metrics-collector

FROM gcr.io/distroless/static:nonroot
LABEL source = git@github.com:kyma-project/control-plane.git

WORKDIR /app

COPY --from=builder /app /app
USER nonroot:nonroot

ENTRYPOINT ["/app/kyma-metrics-collector"]
