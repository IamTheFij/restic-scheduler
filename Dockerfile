ARG ALPINE_VERSION=3.22

FROM golang:alpine${ALPINE_VERSION} AS nomad-build

RUN apk add git make bash build-base linux-headers && \
        git clone --depth 1 --branch v1.10.2 https://github.com/hashicorp/nomad.git /nomad

WORKDIR /nomad
RUN make bootstrap dev && rm -fr /nomad/pkg /nomad/vendor

FROM alpine:${ALPINE_VERSION}

RUN apk add --no-cache \
        bash~=5 \
        mariadb-client~=11 \
        mariadb-connector-c~=3 \
        postgresql17-client~=17 \
        rclone~=1.69 \
        redis~=8 \
        restic~=0.18 \
        sqlite~=3 \
        tzdata~=2025 \
        ;

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/restic-scheduler-$TARGETOS-$TARGETARCH /bin/restic-scheduler

# Install nomad
COPY --from=nomad-build /nomad/bin/nomad /bin/
COPY --from=hashicorp/consul:1.21 /bin/consul /bin/

HEALTHCHECK CMD ["wget", "-O", "-", "http://localhost:8080/health"]

ENTRYPOINT [ "/bin/restic-scheduler" ]
