FROM debian:bookworm-slim

WORKDIR /app
COPY worker .
EXPOSE 8080
CMD ["./worker"]