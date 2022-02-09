FROM alpine:3.15.0
LABEL source=git@github.com:kyma-project/control-plane.git

ARG MIGRATE_VER=4.15.1

WORKDIR /migrate

SHELL ["/bin/ash", "-eo", "pipefail", "-c"]

RUN apk --no-cache add --update openssl bash --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main
RUN apk --no-cache add postgresql-client
RUN apk --no-cache add --update curl --repository=http://dl-cdn.alpinelinux.org/alpine/v3.14/main
RUN wget --progress=dot:giga "https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VER}/migrate.linux-amd64.tar.gz" -O - | tar -xz
RUN mv migrate /usr/local/bin/migrate

COPY ./migrations/ ./migrations
COPY ./run.sh ./run.sh

ENTRYPOINT ["./run.sh"]
