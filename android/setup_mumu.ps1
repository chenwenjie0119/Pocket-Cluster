# Pocket-Cluster MuMu Android Worker 一键配置
# 前置：已编译 x86_64 的 llama-rpc-server 放在 android/ 目录下
$ErrorActionPreference = 'Stop'
$adb = 'adb'
$bin = 'llama-rpc-server'
$remote = "/data/local/tmp/$bin"

Write-Host '[1/5] 检查 adb 设备...'
& $adb devices

Write-Host '[2/5] 推送 rpc-server 与 keepalive 到模拟器...'
if (-not (Test-Path "./$bin")) {
  Write-Warning "未找到本地 $bin，请先按 docs/MVP_GUIDE.md 交叉编译 x86_64 版本放到 android/ 目录"
} else {
  & $adb push "./$bin" $remote
  & $adb push "./keepalive/keepalive.sh" "/data/local/tmp/keepalive.sh"
  & $adb shell "chmod +x $remote /data/local/tmp/keepalive.sh"
}

Write-Host '[3/5] 系统层豁免（防杀）...'
& $adb shell "settings put global cached_apps_freezer disabled"
& $adb shell "dumpsys deviceidle disable"
& $adb shell "cmd appops set com.termux RUN_IN_BACKGROUND allow"
& $adb shell "am set-standby-bucket com.termux active"

Write-Host '[4/5] 端口转发（宿主 50054 -> 模拟器 50052）...'
& $adb forward tcp:50054 tcp:50052

Write-Host '[5/5] 启动 worker 与 keepalive...'
& $adb shell "sh -c 'nohup $remote --host 0.0.0.0 --port 50052 >/data/local/tmp/rpc.log 2>&1 &'"
& $adb shell "sh -c 'nohup sh /data/local/tmp/keepalive.sh >/dev/null 2>&1 &'"

Write-Host '完成。验证: adb shell pidof llama-rpc-server'
