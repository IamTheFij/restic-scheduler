FROM alpine:3.17

RUN apk add --no-cache \
        bash~=5 \
        consul~=1.14 \
        mariadb-client~=10.6 \
        mariadb-connector-c~=3.3 \
        nomad~=1.4 \
        postgresql15-client~=15.3 \
        rclone~=1.60 \
        redis~=7.0 \
        restic~=0.14 \
        sqlite~=3 \
        tzdata~=2023c \
        ;

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/resticscheduler-$TARGETOS-$TARGETARCH /bin/resticscheduler

ENTRYPOINT [ "/bin/resticscheduler" ]
