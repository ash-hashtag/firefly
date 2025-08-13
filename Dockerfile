FROM golang:1.24.6-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o app .

FROM scratch
COPY --from=builder /src/app /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV PORT=37373
EXPOSE $PORT

ENTRYPOINT ["/app"]
