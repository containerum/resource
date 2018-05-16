FROM golang:1.9-alpine as builder
WORKDIR /go/src/git.containerum.net/ch/resource-service
COPY . .
RUN go build -v -ldflags="-w -s" -tags "jsoniter" -o /bin/resource-service ./cmd/resource-service

FROM alpine:3.7
RUN mkdir -p /app
COPY --from=builder /bin/resource-service /app
ENV DB_URL="postgres://user:password@localhost:5432/resource_service?sslmode=disable" \
    KUBE_ADDR=""
EXPOSE 1213
CMD "/app/resource-service"
