FROM golang:1.10-alpine as builder
RUN apk add --update make git
WORKDIR /go/src/git.containerum.net/ch/resource
COPY . .
RUN VERSION=$(git describe --abbrev=0 --tags) make build-for-docker

FROM alpine:3.7

COPY --from=builder /tmp/resource /
ENV DEBUG="true" \
    TEXTLOG="true" \
    MONGO_ADDR="http://mongo:27017" \
    MIN_SERVICE_PORT=30000 \
    MAX_SERVICE_PORT=32767
EXPOSE 1213

CMD "/resource"
