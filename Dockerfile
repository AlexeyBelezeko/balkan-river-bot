FROM golang:1.24-alpine AS build

# Install build dependencies for CGO
RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /build && \
    CGO_ENABLED=1 go build -o /build/water-bot cmd/bot/bot.go && \
    CGO_ENABLED=1 go build -o /build/water-scrapper cmd/scrapper/scrapper.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates && apk add --no-cache tzdata
WORKDIR /app/
RUN mkdir -p data
COPY --from=build /build/water-bot /app/
COPY --from=build /build/water-scrapper /app/
# Create directory for sqlite database
RUN mkdir -p /app/data && chmod 777 /app/data
# Default to running the bot, can be overridden with command
CMD ["./water-bot"]