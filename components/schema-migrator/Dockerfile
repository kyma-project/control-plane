FROM golang:1.18.3-alpine3.15 AS build

COPY cleaner /program/cleaner
COPY go.mod /program/go.mod
COPY main.go /program/main.go

WORKDIR /program
RUN go mod tidy && go mod vendor && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/program /program/main.go

WORKDIR /migrate

FROM scratch

LABEL source=git@github.com:kyma-project/control-plane.git

ARG MIGRATE_VER=4.15.2

WORKDIR /migrate

COPY --from=build /bin/program /bin/program
COPY ./migrations/ ./migrations

ENTRYPOINT ["/bin/program"]
