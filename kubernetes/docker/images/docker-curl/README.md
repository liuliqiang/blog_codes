# Curl Docker Image

## Description

Small docker image with curl based on [Alpine Linux](https://hub.docker.com/_/alpine/).
The curl version is new, so the option `--unix-socket` for Docker API requests is available.

[![](https://images.microbadger.com/badges/image/pstauffer/curl.svg)](https://microbadger.com/images/pstauffer/curl)

## Usage
```
docker run -d --name curl pstauffer/curl:latest
docker run --rm --name curl pstauffer/curl:latest curl --version
docker run --rm --name curl pstauffer/curl:latest curl http://www.google.ch
```

## License
This project is licensed under `MIT <http://opensource.org/licenses/MIT>`_.
