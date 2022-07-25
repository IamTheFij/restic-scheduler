FROM alpine:3

RUN apk add --no-cache consul~=1.12 redis~=7.0 mariadb-client~=10.6 mariadb-connector-c~=3.1 sqlite~=3 bash~=5 restic~=0.13

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/resticscheduler-$TARGETOS-$TARGETARCH /bin/resticscheduler

ENTRYPOINT [ "/bin/resticscheduler" ]
