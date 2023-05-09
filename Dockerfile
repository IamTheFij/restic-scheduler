FROM alpine:3.17

RUN apk add --no-cache \
        bash~=5 \
        consul~=1.14 \
        nomad~=1.4 \
        mariadb-client~=10.6 \
        mariadb-connector-c~=3.3 \
        rclone~=1.60 \
        redis~=7.0 \
        restic~=0.14 \
        sqlite~=3 \
        ;

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/resticscheduler-$TARGETOS-$TARGETARCH /bin/resticscheduler

ENTRYPOINT [ "/bin/resticscheduler" ]
