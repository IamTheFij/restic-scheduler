FROM scratch

ARG BIN=./dist/resticscheduler-linux-amd64
COPY ${BIN} /bin/resticscheduler

ENTRYPOINT [ "/resticscheduler" ]
