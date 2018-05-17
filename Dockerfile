FROM golang:1.9-alpine as builder
WORKDIR /go/src/git.containerum.net/ch/resource-service
COPY . .
RUN go build -v -ldflags="-w -s" -tags "jsoniter" -o /bin/resource-service ./cmd/resource-service

FROM alpine:3.7
RUN mkdir -p /app
COPY --from=builder /bin/resource-service /app
ENV CH_RESOURCE_DEBUG="true" \
    KUBE_ADDR="" \
    CH_RESOURCE_TEXTLOG="true"
EXPOSE 1213
CMD "/app/resource-service"
