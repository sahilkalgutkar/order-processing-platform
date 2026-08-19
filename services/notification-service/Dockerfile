FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/notification-service ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/notification-service /notification-service
EXPOSE 8082
ENTRYPOINT ["/notification-service"]
