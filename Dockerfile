FROM  golang:1.18 as builder
ENV GO111MODULE=on

WORKDIR /workspace

COPY ./ ./

RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/go \
    go build -o bot-bin .

FROM centos:7
ENV LANG=en_US.utf8

COPY --from=builder /workspace/bot-bin /tmp/bot-bin

CMD ["/tmp/bot-bin"]