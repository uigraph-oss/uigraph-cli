FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /uigraph-cli .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git
COPY --from=builder /uigraph-cli /usr/local/bin/uigraph-cli
ENTRYPOINT ["/usr/local/bin/uigraph-cli"]
