FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO (needed for go-sqlite3)
RUN apk add --no-cache gcc musl-dev

COPY . .
# Run tidy to generate go.sum and download dependencies
RUN go mod tidy

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/server .

# Create data directory
RUN mkdir -p ./data

EXPOSE 8080

CMD ["./server"]
