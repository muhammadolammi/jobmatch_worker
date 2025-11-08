FROM debian:bookworm-slim

WORKDIR /app
# Install CA certificates so Go can verify TLS
RUN apt-get update && apt-get install -y ca-certificates

COPY worker .
EXPOSE 8080
CMD ["./worker"]