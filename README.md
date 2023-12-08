# AthenaBot Project

## Deploy your own bot

```shell
docker run --name redis -v /data/redis:/data -d redis:5

docker build -t="athenabot:latest" .

docker run --name athenabot \
-e BOT_CONFIG="./config.json" \
-d athenabot:latest
```