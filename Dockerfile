FROM golang:latest
COPY . /usr/src/fifo
WORKDIR /usr/src/fifo
RUN sh /usr/src/fifo/build.sh -o /tmp/fifo

FROM alpine:latest
RUN apk add ca-certificates
COPY --from=0 /tmp/fifo /bin/fifo
ENTRYPOINT ["/bin/fifo"]