FROM eu.gcr.io/kyma-project/test-infra/buildpack-golang-toolbox:v20210125-6234473e as builder

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
