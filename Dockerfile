FROM alpine:3

RUN apk add --no-cache mysql-client~=10.6 sqlite~=3 bash~=5

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/resticscheduler-$TARGETOS-$TARGETARCH /bin/resticscheduler

ENTRYPOINT [ "/bin/resticscheduler" ]
