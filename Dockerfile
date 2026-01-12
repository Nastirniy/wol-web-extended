FROM docker.io/library/golang:1.24 AS server_builder
WORKDIR /app
COPY ./apps/server /app
RUN CGO_ENABLED=0 GOOS=linux go build -o wol-server .


FROM docker.io/oven/bun:1 AS web_builder
WORKDIR /app
COPY . .
RUN bun install
RUN bun run build
# Copy post-build fix to server static
# Note: This ensures CSS links injected by postbuild.ts are included in the image.
# For future Docker Hub releases, ensure this step is preserved to maintain UI styling.
WORKDIR /app/apps/web
RUN bun run postbuild


FROM docker.io/library/alpine:latest
WORKDIR /app

# Install iproute2 for ip command (ARP cache management)
RUN apk add --no-cache iproute2

# Binaries from build stages
COPY --from=server_builder /app/wol-server ./wol-server
COPY --from=web_builder /app/apps/server/static ./static

# Server listen address (overrides config.json)
ENV LISTEN_ADDRESS=:8090
EXPOSE 8090

# Run with config and db in /app/data (persistent volume)
CMD ["/app/wol-server", "-config", "/app/data/config.json", "-db", "/app/data/wol.db"]
