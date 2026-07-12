FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /qg-server ./cmd/qg-server && \
    CGO_ENABLED=0 go build -o /qg ./cmd/qg

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends python3 ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /qg-server /usr/local/bin/qg-server
COPY --from=build /qg /usr/local/bin/qg
COPY qualiguard.yaml /app/qualiguard.yaml
ENV QG_DATA_DIR=/data
VOLUME /data
EXPOSE 9000
ENTRYPOINT ["qg-server", "--host", "0.0.0.0", "--port", "9000", "--data-dir", "/data", "--work-dir", "/app", "--config", "/app/qualiguard.yaml"]
