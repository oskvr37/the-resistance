FROM golang:1.25.5-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main ./cmd/server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8000
CMD ["./main"]