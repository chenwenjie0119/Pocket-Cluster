# 排障指南

## WSL2 / Docker

| 症状 | 原因 | 解决 |
|---|---|---|
| wsl 内存配置不生效 | 未重启 WSL | `wsl --shutdown` 后重新打开 |
| docker stats 显示内存超 1536m | swap 逃逸 | 检查 compose 是否写了 memswap_limit |
| 容器间 ping 不通 | 网络驱动问题 | 确认 compose 里同一个 bridge 网络 |
| Docker Desktop 占用过高 | WSL2 后端吃配置 | 检查 .wslconfig；确认 Settings 里未开"资源占用大"选项 |

## MuMu / Android

| 症状 | 原因 | 解决 |
|---|---|---|
| adb forward 后 curl 不通 | 模拟器内进程没起来 | `adb shell pidof llama-rpc-server`；无则手动拉起 |
| 进程反复被杀 | 系统后台限制 | 执行 android/keepalive 的 adb 豁免命令；MuMu 设置关闭后台清理 |
| MuMu 内存暴涨 | 模拟器分配过大 | 设置里降到 2048MB，分辨率降到 720p |
| `adb shell input` 无响应 | MuMu input 通道不稳定 | 改用 sendevent 或手动点击（已知坑） |
| vmindex 对不上 | MuMu 多开索引是枚举序号 | 用 `mumu-cli info --vmindex all` 核对 adb 端口对应 index 再操作 |

## 推理 / llama.cpp

| 症状 | 原因 | 解决 |
|---|---|---|
| --rpc 连不上 | 端口/防火墙 | Windows 放行 TCP 50052-50054；容器确认 EXPOSE |
| 推理极慢 | MuMu NAT 转发瓶颈 | 把 MuMu 的 -ts 比例调低（如 0.1~0.2）或只跑 Docker 双 worker |
| OOM | 模型超预算 | 换更小量化（Q4→Q3/IQ4）；降低 n_ctx；检查 mem_free 后重新 plan |
| -ts 比例失效 | 语法错误 | 用 `-ts 0.4,0.4,0.2` 逗号分隔、总和为 1 |
| 张量切分日志看不到 | 版本过旧 | 升级 llama.cpp（需 >= 2024 年 RPC 支持版本） |

## 内存红线相关

| 症状 | 原因 | 解决 |
|---|---|---|
| watchdog 没有生效 | 未以管理员运行 / 计划任务未配 | 右键管理员运行；或注册计划任务开机自启 |
| 90% 时仍卡死 | 基线内存已高 | 启动集群前关闭 Chrome/Edge 多标签、更新服务 |
