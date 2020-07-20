FROM golang:1.14.4-alpine3.12 as build

ARG VERSION
ARG COMMITHASH
ARG CGO_ENABLED
ARG GO111MODULE

ENV VERSION=${VERSION:-dev}
ENV COMMITHASH=${COMMITHASH:-dev}
ENV CGO_ENABLED=${CGO_ENABLED:-0}
ENV GO111MODULE=${GO111MODULE:-on}
ENV BASE_APP_DIR=/go/src/github.com/kyma-project/control-plane/components/metris

WORKDIR ${BASE_APP_DIR}

COPY . .

RUN go build -v -a -o /bin/metris \
  -ldflags "-s -w -extldflags '-static'" \
  -ldflags "-X github.com/kyma-project/control-plane/components/metris/internal/version.Version=${VERSION}" \
  -ldflags "-X github.com/kyma-project/control-plane/components/metris/internal/version.CommitHash=${COMMITHASH}" \
  ./cmd/metris/main.go

# https://github.com/GoogleContainerTools/distroless/blob/master/base/README.md
FROM gcr.io/distroless/static-debian10:nonroot
LABEL source=git@github.com:kyma-project/control-plane.git

COPY --from=build /bin/metris /bin/metris

ENTRYPOINT ["/bin/metris"]