# Pocket-Cluster Worker: llama.cpp rpc-server (Linux CPU)
# 构建参数 -DGGML_RPC=ON 启用 RPC 张量并行后端（新版 llama.cpp 的 RPC 已并入 ggml）。
# 前置：llama.cpp 源码不入库，需先在 docker/ 下 clone 一份（见 README 快速开始）。
FROM ubuntu:22.04 AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY llama.cpp/ /llama.cpp/

RUN cd /llama.cpp \
    && cmake -B build -DGGML_RPC=ON -DBUILD_SHARED_LIBS=ON -DCMAKE_BUILD_TYPE=Release \
    && cmake --build build --config Release -j "$(nproc)" --target ggml-rpc-server

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libgomp1 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /llama.cpp/build/bin/ggml-rpc-server /usr/local/bin/llama-rpc-server
COPY --from=build /llama.cpp/build/bin/libggml* /usr/local/lib/
ENV LD_LIBRARY_PATH=/usr/local/lib
RUN ldconfig
COPY worker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 50052
ENTRYPOINT ["/entrypoint.sh"]
