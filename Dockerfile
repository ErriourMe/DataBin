FROM golang:1.27.3-alpine AS builder

ENV CGO_ENABLED=0 \
    GO111MODULE=on

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags "-s -w" -o main .

FROM alpine:3.22.2

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /www

COPY --from=builder /build/main /www/main
COPY --from=builder /build/version.txt /www/version.txt
COPY --from=builder /build/public/ /www/public/
COPY --from=builder /build/resources/ /www/resources/

ENV APP_HOST=0.0.0.0 \
    APP_PORT=3000

EXPOSE 3000

ENTRYPOINT ["/www/main"]
