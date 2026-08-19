FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/inventory-service ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /out/inventory-service /inventory-service
EXPOSE 8081
ENTRYPOINT ["/inventory-service"]
