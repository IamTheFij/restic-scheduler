FROM alpine:3

RUN apk add --no-cache mariadb-client~=10.6 mariadb-connector-c~=3.1 sqlite~=3 bash~=5 restic~=0.12

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/resticscheduler-$TARGETOS-$TARGETARCH /bin/resticscheduler

ENTRYPOINT [ "/bin/resticscheduler" ]
