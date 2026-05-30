FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY backend/go.mod ./
COPY backend ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/alatube ./cmd/alatube

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates docker.io \
  && rm -rf /var/lib/apt/lists/* \
  && useradd --uid 10000 --create-home --shell /usr/sbin/nologin alatube
USER alatube
WORKDIR /app
COPY --from=build /out/alatube /app/alatube
EXPOSE 8080
ENTRYPOINT ["/app/alatube"]

