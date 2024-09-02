FROM rust:1.72 as builder

WORKDIR /usr/src/app

COPY . .

RUN cargo build --release

COPY . .


FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y libssl-dev && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /usr/src/app

COPY --from=builder /usr/src/app/target/release/calendar-backend .

EXPOSE 8080

CMD ["./calendar-backend"]
