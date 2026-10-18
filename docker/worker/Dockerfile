# Pocket-Cluster Worker: llama.cpp rpc-server (Linux CPU)
# 构建参数 -DLLAMA_RPC=ON 启用 RPC 张量并行后端。
FROM ubuntu:22.04 AS build

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential cmake git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 https://github.com/ggml-org/llama.cpp.git /llama.cpp

RUN cd /llama.cpp \
    && cmake -B build -DLLAMA_RPC=ON -DCMAKE_BUILD_TYPE=Release \
    && cmake --build build --config Release -j "$(nproc)" --target rpc-server

FROM ubuntu:22.04
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /llama.cpp/build/bin/llama-rpc-server /usr/local/bin/llama-rpc-server
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 50052
ENTRYPOINT ["/entrypoint.sh"]
