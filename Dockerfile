FROM golang:1.9-alpine as builder
WORKDIR /go/src/git.containerum.net/ch/resource-service
COPY . .
RUN go build -v -ldflags="-w -s" -tags "jsoniter" -o /bin/resource-service

FROM alpine:3.7
COPY --from=builder /bin/resource-service /app
COPY --from=builder /go/src/git.containerum.net/ch/resource-service/migrations /app/migrations
ENV MIGRATION_URL="file:///app/migrations" \
    DB_URL="postgres://user:password@localhost:5432/resource_service?sslmode=disable" \
    MODE="release" \
    AUTH_ADDR="" \
    BILLING_ADDR="" \
    KUBE_ADDR="" \
    MAILER_ADDR="" \
    VOLUMES_ADDR="" \
    USER_ADDR="" \
    LISTEN_ADDR=""
EXPOSE 1213
ENTRYPOINT ["/app/resource-service"]
