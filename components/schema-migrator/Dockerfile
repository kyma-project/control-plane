FROM alpine:3.12.3
LABEL source=git@github.com:kyma-project/control-plane.git

ARG MIGRATE_VER=4.5.0

WORKDIR /migrate
RUN apk --no-cache upgrade && \
    apk --no-cache add bash postgresql-client &&\
    apk --no-cache add --update curl --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main
RUN wget https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VER}/migrate.linux-amd64.tar.gz -O - | tar -xz
RUN mv migrate.linux-amd64 /usr/local/bin/migrate

COPY ./migrations/ ./migrations
COPY ./run.sh ./run.sh

ENTRYPOINT ["./run.sh"]
