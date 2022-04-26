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
RUN /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
RUN echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> /root/.profile
RUN eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"

RUN /bin/bash -c "$(git clone https://github.com/Homebrew/brew ~/.linuxbrew/Homebrew)"
RUN /bin/bash -c "$(mkdir ~/.linuxbrew/bin)"
RUN /bin/bash -c "$(ln -s ../Homebrew/bin/brew ~/.linuxbrew/bin)"
RUN /bin/bash -c "$(eval $(~/.linuxbrew/bin/brew shellenv))"
RUN brew --version

RUN brew install int128/kubelogin/kubelogin
RUN go build -v -o main ./cmd/generator/main.go
RUN mkdir /app && mv ./main /app/main && mv ./licenses /app/licenses

FROM eu.gcr.io/kyma-project/external/alpine:3.14.2
LABEL source = git@github.com:kyma-project/control-plane.git

WORKDIR /app

#
# Install certificates
#

RUN apk add --no-cache ca-certificates openssl

#
# Copy binary
#

COPY --from=builder /app /app

#
# Run app
#

CMD ["/app/main"]
