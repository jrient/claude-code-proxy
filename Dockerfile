# Minimal runtime - binary is mounted from host
FROM debian:bookworm-slim
WORKDIR /app
EXPOSE 8080 8081
VOLUME /app/data
CMD ["./claude-code-proxy"]
