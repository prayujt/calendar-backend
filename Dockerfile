FROM golang:1.21 AS builder

ARG TARGETARCH=arm64

WORKDIR /usr/src/app

COPY . .

ENV GOTOOLCHAIN=auto
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -o calendar-backend ./cmd/calendar-backend


FROM debian:bookworm-slim

WORKDIR /usr/src/app

COPY --from=builder /usr/src/app/calendar-backend .

EXPOSE 8080

CMD ["./calendar-backend"]
