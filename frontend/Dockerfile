FROM node:24-bookworm AS build
WORKDIR /src
COPY package.json package-lock.json* ./
COPY frontend/package.json frontend/package.json
RUN npm install
COPY frontend frontend
RUN npm --workspace frontend run build

FROM caddy:2.8-alpine
COPY --from=build /src/frontend/build /srv
COPY deploy/caddy/Caddyfile /etc/caddy/Caddyfile

