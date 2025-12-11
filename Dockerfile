FROM ubuntu:latest

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        bash \
        mariadb-client \
        libmariadb3 \
        postgresql-client \
        rclone \
        redis-tools \
        restic \
        sqlite3 \
        tzdata \
    && rm -rf /var/lib/apt/lists/*

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/restic-scheduler-$TARGETOS-$TARGETARCH /bin/restic-scheduler

# Install nomad
COPY --from=hashicorp/nomad:1.11 /bin/nomad /bin/
COPY --from=hashicorp/consul:1.22 /bin/consul /bin/

HEALTHCHECK CMD ["wget", "-O", "-", "http://localhost:8080/health"]

ENTRYPOINT [ "/bin/restic-scheduler" ]
