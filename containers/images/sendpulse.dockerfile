FROM golang:1.24-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git make

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build && mv build/sendpulse sendpulse

FROM alpine:latest
WORKDIR /src

LABEL maintainer="Bora Tanrikulu <me@bora.sh>"

RUN apk --no-cache add ca-certificates wget

COPY --from=builder /app/sendpulse /bin/sendpulse
COPY scripts/entrypoint.sh /bin/entrypoint.sh

RUN chmod +x /bin/entrypoint.sh

ENTRYPOINT ["/bin/entrypoint.sh"]
