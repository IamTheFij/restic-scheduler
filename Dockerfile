FROM alpine:3

RUN apk add --no-cache \
        bash~=5 \
        consul~=1.12 \
        mariadb-client~=10.6 \
        mariadb-connector-c~=3.1 \
        rclone~=1.58 \
        redis~=7.0 \
        restic~=0.13 \
        sqlite~=3 \
        ;

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/resticscheduler-$TARGETOS-$TARGETARCH /bin/resticscheduler

ENTRYPOINT [ "/bin/resticscheduler" ]
