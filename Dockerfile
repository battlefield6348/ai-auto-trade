FROM golang:1.24-alpine AS build
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/ai-auto-trade ./cmd/api

FROM alpine:3.19 AS runtime
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /app/bin/ai-auto-trade /app/ai-auto-trade
EXPOSE 8080
ENV HTTP_ADDR=:8080
ENTRYPOINT ["/app/ai-auto-trade"]
