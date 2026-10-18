# Pocket-Cluster Master 容器化（可选）
# 推荐：Master 直接跑 Windows 原生二进制更省内存（~20MB）；
# 需要容器化部署时使用本镜像。
FROM golang:1.21-alpine AS build
WORKDIR /src
COPY master/ ./master/
WORKDIR /src/master
RUN go mod download && CGO_ENABLED=0 go build -o pocketd ./cmd/pocketd

FROM alpine:3.19
COPY --from=build /src/master/pocketd /usr/local/bin/pocketd
EXPOSE 8080
ENTRYPOINT ["pocketd"]
