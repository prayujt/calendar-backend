FROM rust:1.72 as builder

WORKDIR /usr/src/app

COPY . .

RUN cargo build --release

COPY . .


FROM debian:bullseye-slim

WORKDIR /usr/src/app

COPY --from=builder /usr/src/app/target/release/calendar-backend .

EXPOSE 8080

CMD ["./calendar-backend"]
