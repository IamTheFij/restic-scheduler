FROM ubuntu:26.04

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        'bash=5.*' \
        'ca-certificates=*' \
        'curl=8.*' \
        'mariadb-client=1:11.*'\
        'libmariadb3=1:11.*' \
        'postgresql-client-18=18.*' \
        'rclone=1.60.*' \
        'redis-tools=5:8.*' \
        'restic=0.18.*' \
        'sqlite3=3.*' \
        'tzdata=202*' \
    && rm -rf /var/lib/apt/lists/*

ARG DIST=dist
ARG TARGETOS
ARG TARGETARCH
COPY ./$DIST/restic-scheduler-$TARGETOS-$TARGETARCH /bin/restic-scheduler

# Install nomad
COPY --from=hashicorp/nomad:2.0 /bin/nomad /bin/
COPY --from=hashicorp/consul:1.22 /bin/consul /bin/

HEALTHCHECK CMD ["curl", "--silent", "--show-error", "--fail", "-o", "/dev/null", "http://localhost:8080/health"]

ENTRYPOINT [ "/bin/restic-scheduler" ]
