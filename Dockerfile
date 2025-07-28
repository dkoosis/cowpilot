FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build everything server
RUN go build -o everything ./cmd/everything
# Build RTM server
RUN go build -o rtm-server ./cmd/rtm-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata sqlite-libs
WORKDIR /root/

COPY --from=builder /app/rtm-server .

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

RUN chmod +x ./rtm-server

CMD ["./rtm-server"]
