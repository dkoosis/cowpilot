FROM golang:1.23-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build server based on SERVER_TYPE arg (defaults to rtm)
ARG SERVER_TYPE=rtm
RUN go build -o server ./cmd/${SERVER_TYPE}

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata sqlite-libs
WORKDIR /root/

COPY --from=builder /app/server .

RUN chmod +x ./server

CMD ["./server"]
