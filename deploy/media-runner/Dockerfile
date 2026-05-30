FROM python:3.12-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
  && rm -rf /var/lib/apt/lists/* \
  && pip install --no-cache-dir yt-dlp \
  && useradd --uid 10001 --create-home --shell /usr/sbin/nologin media

USER media
WORKDIR /work
ENTRYPOINT []

