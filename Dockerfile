FROM golang:1.9-alpine as builder
WORKDIR /go/src/git.containerum.net/ch/resource-service
COPY . .
RUN CGO_ENABLED=0 go build -v -ldflags="-w -s -extldflags '-static'" -o /bin/resource-service

FROM scratch
COPY --from=builder /bin/resource-service /
COPY --from=builder /go/src/git.containerum.net/ch/resource-service/migrations /migration
ENV MIGRATION_URL="file:///migration" \
    DB_URL="postgres://user:password@localhost:5432/resource_service?sslmode=disable" \
    MODE="release" \
    AUTH_ADDR="" \
    BILLING_ADDR="" \
    KUBE_ADDR="" \
    MAILER_ADDR="" \
    VOLUMES_ADDR="" \
    LISTEN_ADDR=""
EXPOSE 1213
ENTRYPOINT ["/resource-service"]
