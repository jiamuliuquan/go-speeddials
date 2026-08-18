# 构建阶段
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o speeddials .

# 运行阶段
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/speeddials .

ENV ADDR=:8080 \
    DATA_DIR=/app/data \
    TZ=Asia/Shanghai

RUN mkdir -p /app/data/uploads

EXPOSE 8080

VOLUME ["/app/data"]

ENTRYPOINT ["./speeddials"]
