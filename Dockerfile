FROM rust:1.72 as builder

WORKDIR /usr/src/app

COPY Cargo.toml Cargo.lock ./

RUN cargo build --release
RUN rm -f target/release/deps/calendar-backend*

COPY . .

RUN cargo build --release


FROM debian:buster-slim

WORKDIR /usr/src/app

COPY --from=builder /usr/src/app/target/release/calendar-backend .

EXPOSE 8080

CMD ["./calendar-backend"]
