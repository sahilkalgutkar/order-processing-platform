FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/order-service ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/order-service /order-service
EXPOSE 8080
ENTRYPOINT ["/order-service"]
