# Stage 1: Build Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o fnos-frpc-gui .

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary (static files are embedded via go:embed)
COPY --from=builder /src/fnos-frpc-gui .

# Data volume
VOLUME /app/data

# Default port (used with host network)
ENV WEB_PORT=7500

ENTRYPOINT ["./fnos-frpc-gui"]
