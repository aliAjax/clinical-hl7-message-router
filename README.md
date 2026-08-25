# HL7 v2 Message Router

企业级纯Go HL7 v2临床消息路由与转换网关。服务通过MLLP或HTTP接收消息，解析并校验消息头，按已发布规则选择目标，记录幂等状态和投递轨迹，并提供失败查询与重放入口。项目只处理临床消息交换，不承载患者档案或诊疗业务。

## 能力

- MLLP TCP监听、半包/粘包读取、并发连接与AA/AE ACK响应
- MSH、EVN、PID、PV1、ORC、OBR、OBX、NTE及未知段保留
- 一至多层重复字段、组件、子组件字段路径读取和更新
- 按消息类型、触发事件、发送设施匹配已发布路由
- 代码表、默认值、日期转换、字段脱敏映射操作
- 基于MSH-10的幂等接收、投递状态、追踪和手工重放
- 重试工作器、指数退避、死信与保留期清理领域实现
- 请求ID、认证、限流、超时、恢复、请求体限制和结构化日志
- 健康、就绪、Prometheus格式指标和优雅停机
- 无依赖运维页面，可探测健康、就绪和指标入口

日志仅记录消息ID、路由、状态和摘要，不记录患者姓名、证件号或完整HL7原文。

## 目录

```text
cmd/router                 HTTP和MLLP服务入口
cmd/mllp-simulator         可配置ACK的模拟MLLP目标
internal/hl7               HL7解析、校验、序列化和脱敏
internal/segment           强类型字段路径和段定义
internal/routing           草稿校验、发布与目标匹配
internal/mapping           字段映射和转换
internal/delivery          连接器、投递状态和退避
internal/deadletter        死信登记
internal/retention         保留策略和批量清理
internal/trace             消息幂等登记和事件轨迹
internal/worker            可取消并发后台作业运行器
api                        OpenAPI和gRPC契约
configs                    YAML配置示例
migrations                 PostgreSQL有序迁移
deploy                     Docker Compose
samples                    HL7和映射样例
scripts                    启动和验证脚本
web                        路由网关运维页面及静态构建脚本
```

## 本地运行

要求Go 1.22或更高版本。

```bash
go test ./...
go build -o bin/hl7-router ./cmd/router
HTTP_ADDR=:8084 MLLP_ADDR=:2575 ./bin/hl7-router
```

运维页面使用 Node.js 20 或更高版本构建：

```bash
npm run build
```

环境变量：`HTTP_ADDR`、`MLLP_ADDR`、`DATABASE_URL`和`AUTH_TOKEN`。当前可执行默认使用进程内适配器，迁移文件定义了生产PostgreSQL持久化模型。

运行快速端到端检查：

```bash
./scripts/verify.sh
```

## HTTP主流程

创建目标：

```bash
curl -sS -H 'Content-Type: application/json' \
  -d '{"ID":"ehr","Name":"EHR","Address":"memory://ehr"}' \
  http://127.0.0.1:8084/v1/targets
```

创建、验证并发布路由：

```bash
curl -sS -H 'Content-Type: application/json' \
  -d '{"ID":"adt","Name":"ADT A01","MessageType":"ADT","Trigger":"A01","TargetIDs":["ehr"]}' \
  http://127.0.0.1:8084/v1/routes
curl -sS -X POST -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8084/v1/routes/adt/validate
curl -sS -X POST -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8084/v1/routes/adt/publish
```

提交消息后，以返回的`id`查询或重放：

```bash
curl -sS -H 'Content-Type: application/json' \
  -d '{"raw":"MSH|^~\\\\&|LAB|HOSP|EHR|HOSP|20260821150000||ADT^A01|MSG-001|P|2.5\rPID|1||12345||DOE^JANE"}' \
  http://127.0.0.1:8084/v1/messages
curl -sS http://127.0.0.1:8084/v1/messages/MSG-001
curl -sS -X POST -H 'Content-Type: application/json' -d '{}' http://127.0.0.1:8084/v1/messages/MSG-001/replay
```

再次提交相同MSH-10会返回`duplicate: true`，不会重复投递。

## MLLP验证

MLLP帧格式为`0x0b + HL7 + 0x1c + 0x0d`。启动模拟目标：

```bash
go run ./cmd/mllp-simulator -listen :3575 -ack AA
```

服务自身监听`:2575`。测试覆盖将一帧拆成多个片段读取的场景；无效消息返回AE，合法消息返回AA。可用`-ack AE`模拟坏ACK目标。

## 可靠性语义

- 接收幂等键优先使用MSH-10；缺失时解析阶段拒绝，避免无法追踪的消息。
- 目标投递状态为`pending`、`delivered`、`failed`或`dead`。
- 后台作业持久化模型使用`FOR UPDATE SKIP LOCKED`安全领取任务。
- 重试采用有上限的指数退避；达到最大次数后进入死信。
- 路由在发布前可验证目标引用，发布后版本递增。
- 原文不进入日志；API查询只返回摘要、状态和事件。

## 容器运行

```bash
docker compose -f deploy/docker-compose.yml up --build
docker compose -f deploy/docker-compose.yml down -v
```

Compose启动PostgreSQL并按文件名顺序加载迁移。停止命令中的`-v`会同时删除本地演示数据卷，请勿用于需要保留数据的环境。

## 验证清单

- `go test ./...`覆盖HL7必填字段、未知段、字段重复、MLLP半包、路由发布、幂等和查询主流程。
- `go build ./cmd/router ./cmd/mllp-simulator`验证两个交付二进制。
- `/healthz`证明进程存活，`/readyz`证明接入已就绪，`/metrics`暴露请求计数。
- `samples/adt-a01.hl7`与`samples/mapping.yaml`用于人工协议检查。
- `api/openapi.yaml`和`api/router.proto`是HTTP与gRPC契约。
