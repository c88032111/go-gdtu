# Build Ggdtu in a stock Go builder container
FROM golang:1.16-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-gdtu
RUN cd /go-gdtu && make ggdtu

# Pull Ggdtu into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-gdtu/build/bin/ggdtu /usr/local/bin/

EXPOSE 8545 8546 30303 30303/udp
ENTRYPOINT ["ggdtu"]
