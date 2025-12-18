FROM ubuntu:rolling

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        'bash=5.*' \
        'mariadb-client=1:11.*'\
        'libmariadb3=1:11.*' \
        'postgresql-client-17=17.*' \
        'rclone=1.60.*' \
        'redis-tools=5:8.*' \
        'restic=0.18.*' \
        'sqlite3=3.*' \
        'tzdata=202*' \
    && rm -rf /var/lib/apt/lists/*

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/restic-scheduler-$TARGETOS-$TARGETARCH /bin/restic-scheduler

# Install nomad
COPY --from=hashicorp/nomad:1.11 /bin/nomad /bin/
COPY --from=hashicorp/consul:1.22 /bin/consul /bin/

HEALTHCHECK CMD ["wget", "-O", "-", "http://localhost:8080/health"]

ENTRYPOINT [ "/bin/restic-scheduler" ]
