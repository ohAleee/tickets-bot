FROM rust:1-bullseye

RUN apt-get install -y apt-transport-https
RUN apt-get update && apt-get -y upgrade && apt-get -y install python3 openssl libssl-dev ca-certificates cmake

RUN mkdir -p /tmp/compile
WORKDIR /tmp/compile

COPY . .

# Public gateway sharder (default features: mimalloc, sentry, metrics, ...).
RUN cargo build --release --bin public

# Prod container
FROM debian:bullseye

RUN apt-get update && apt-get -y upgrade && apt-get -y install openssl libssl-dev ca-certificates

COPY --from=0 /tmp/compile/target/release/public /srv/sharder/public
RUN chmod +x /srv/sharder/public

RUN useradd -m container
USER container
WORKDIR /srv/sharder

CMD ["/srv/sharder/public"]
