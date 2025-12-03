FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/ai-auto-trade ./cmd/api

FROM gcr.io/distroless/static-debian12 AS runtime
WORKDIR /app
COPY --from=build /app/bin/ai-auto-trade /app/ai-auto-trade
EXPOSE 8080
ENV HTTP_ADDR=:8080
ENTRYPOINT ["/app/ai-auto-trade"]
