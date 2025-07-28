FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build server based on SERVER_TYPE arg (defaults to rtm)
ARG SERVER_TYPE=rtm
RUN go build -o server ./cmd/${SERVER_TYPE}-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata sqlite-libs
WORKDIR /root/

COPY --from=builder /app/server .

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

RUN chmod +x ./server

CMD ["./server"]
