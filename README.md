# Athena BOT

## config example
conf/example-config.json

## deploy

```bash
docker run --restart always --name redis -v /data/redis:/data -d redis:5

docker build -t="athenabot:latest" .

docker run --restart always --name athenabot \
-e BOT_CONFIG="./config.json" \
-d athenabot:latest
```