FROM golang:1.24-alpine AS build
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ai-auto-trade ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o migrate ./cmd/migrate

FROM alpine:3.19 AS runtime
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
# 將執行檔複製到 PATH 路徑下
COPY --from=build /app/ai-auto-trade /usr/local/bin/ai-auto-trade
COPY --from=build /app/migrate /usr/local/bin/migrate
# 複製前端靜態資源與 Migration 檔案
COPY --from=build /app/web ./web
COPY --from=build /app/db/migrations ./db/migrations
COPY --from=build /app/entrypoint.sh ./entrypoint.sh
# 複製基礎配置
COPY --from=build /app/config.example.yaml ./config.yaml

RUN chmod +x ./entrypoint.sh

EXPOSE 8080
ENV HTTP_ADDR=:8080
ENTRYPOINT ["./entrypoint.sh"]
