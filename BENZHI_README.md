基于 Go 和 Web 实现的 HL7 v2 临床消息路由 Web 后端服务项目，用于接收、转换、投递并追踪医院系统之间的协议消息。

# clinical-hl7-message-router__005

## 构建镜像

请从**仓库根目录**执行；`benzhi.Dockerfile`、`build_benzhi_docker.sh`、`BENZHI_README.md` 均固定在该目录：

```bash
./build_benzhi_docker.sh <image-name> [linux/amd64|linux/arm64]
```

## 标准命令

```bash
go build ./...     # 编译
go run ./cmd/router   # 启动
go test ./...      # 测试（如有）
```

```bash
cd . && npm install   # 前端依赖（镜像构建阶段已预装）
cd . && npm run build   # 构建前端
```

## 环境

- 基础镜像: golang:1.22
- Go 模块目录: `.`
- 依赖已在镜像构建阶段预下载，容器内离线可用。
- 容器内工作目录: `/app`
- 前端目录: `.`（Node.js 20，npm 依赖已在镜像构建阶段预下载）
