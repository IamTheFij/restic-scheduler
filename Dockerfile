FROM alpine:3.18

RUN apk add --no-cache \
        bash~=5 \
        consul~=1 \
        mariadb-client~=10 \
        mariadb-connector-c~=3 \
        nomad~=1 \
        postgresql15-client~=15 \
        rclone~=1.62 \
        redis~=7 \
        restic~=0.15 \
        sqlite~=3 \
        tzdata~=2023 \
        ;

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/resticscheduler-$TARGETOS-$TARGETARCH /bin/resticscheduler

HEALTHCHECK CMD ["wget", "-O", "-", "http://localhost:8080/health"]

ENTRYPOINT [ "/bin/resticscheduler" ]
