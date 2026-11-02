FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app/minecrafter .

FROM alpine:3.24
RUN adduser -D -u 10001 minecrafter
WORKDIR /app
COPY --from=builder /app/minecrafter ./minecrafter
USER minecrafter
ENV MIGRATE="TRUE"
ENTRYPOINT ["./minecrafter"]