FROM scratch

ARG TARGETOS
ARG TARGETARCH
COPY ./dist/resticscheduler-$TARGETOS-$TARGETARCH /bin/resticscheduler

ENTRYPOINT [ "/resticscheduler" ]
