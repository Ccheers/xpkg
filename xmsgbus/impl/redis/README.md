# Redis 消息总线：索引、原子性与升级

维护者：xmsgbus 维护者。更新：2026-11-10；修改存储协议或发布新版本前复核。

> 本版本使用 `hmsgbus:v4:*`。与 v3 的路由、队列和 ACK **不互通，不能直接混合滚动发布**。

## 验收条件

- [ ] 订阅巡检与 ACK 恢复不执行 `KEYS` 或全库 `SCAN`。
- [ ] 续租、最后订阅清理、领取、确认、恢复均经 Redis 原子脚本完成。
- [ ] channel 删除重建后，旧发布快照、旧领取请求、旧回执不影响新生命周期。
- [ ] 错误类型／损坏回执不会被当成正常结果，也不会阻止其他健康回执恢复。
- [ ] Redis v8/v9 适配和隔离 Redis 并发测试通过。
- [ ] 上线前完成下面的停写、排空与切换步骤。

## 调用方式

业务 `IMsgBus`、Publisher、Subscriber 的接口保持原形状；装配接口有变化：

```go
bus := redisbus.NewMsgBusV9(client)
manager := xmsgbus.NewTopicManager(ctx, bus)
publisher := xmsgbus.NewPublisher[*Event](bus, manager, otelOptions)
subscriber := xmsgbus.NewSubscriber[*Event](
    topic, channel, bus, otelOptions, manager,
    xmsgbus.WithHandleFunc(handle),
)
```

- `redisbus` 指 `github.com/ccheers/xpkg/xmsgbus/impl/redis`；v8 使用 `NewMsgBus`。
- 不再给 `NewTopicManager` 传 `SharedStorage` 与 CAS 锁。它接受 `SubscriptionRegistry`，Redis、memory 的 `MsgBus` 均实现该接口。
- `NewTopicManager` 返回 `*TopicManager`，仍满足 `ITopicManager`。
- Redis 和 memory 的 `NewMsgBus` 返回具体类型。将构造函数保存为固定函数类型的调用方需要同步修改。
- 自定义 `core.IRedisClient` adapter 必须实现新增的 `Eval`。
- Kafka 的传输和存储实现不在此次修改范围；它目前没有实现新的 `SubscriptionRegistry`。依赖旧 manager 构造函数的 Kafka 装配需要另行适配，不能声称无影响升级。
- topic、channel、消费者身份不能为空；订阅 TTL 必须为正，Redis 存储按毫秒向上取整。每个消费者使用自己的 UUID。
- 退出时先取消消费者和 manager 的 context，等业务处理退出，再调用 `bus.Close()`，最后关闭 Redis client。`Close` 只停止本 bus 的恢复循环，不关闭调用方持有的 Redis client。

自定义消费者需将 `errors.Is(err, xmsgbus.ErrPopTimeout)` 与 `errors.Is(err, xmsgbus.ErrNoData)` 视为正常空轮询／channel 生命周期结束；不能再只比较 `redis.Nil`。

## 数据与原子操作

| 数据 | 形状 |
|---|---|
| topic 路由 | HASH：channel → generation |
| channel 订阅 | ZSET：消费者 UUID → 到期毫秒 |
| ready 队列 | LIST：原始消息字节 |
| pending 数据 | HASH：receipt UUID → catalog、channel、generation、ready key、消息字节 |
| pending 到期索引 | ZSET：receipt UUID → 到期毫秒 |

topic/channel 用编码后的独立分量组成 key，避免名称中冒号造成身份碰撞。租约时间由 Redis `TIME` 计算。

### 订阅

注册脚本同时更新订阅租约和路由。已有活 channel 的正常积压不被清空。注销只移除指定 UUID；最后一个活订阅消失时，在同一个脚本内删除路由、订阅索引和 ready 队列。周期清理执行同样的判断，并复核查询快照中的 generation。

查询失败时不进行失活清理。错误 key 类型在任何写入之前校验，不能留下“路由已删除、队列还在”的部分操作。

### 发布与领取

发布可先读取 channel 清单，但实际入队脚本必须复核 generation。跨多个 channel 的整次广播不是一个事务，部分失败后重试可能重复投递。

领取不再先 `BLPOP` 再登记 ACK。短脚本在一次原子操作中读取队首、写入 pending 数据及到期索引、移除队首；每次 `Pop` 固定一个 receipt，重发同一领取请求不会额外出队。消息支持空正文与二进制。

空队列时 `Pop` 最多每 100ms 再试一次，受调用 context 和 `blockTimeout` 控制；这是用少量轮询换取原子领取，不再依赖阻塞出队后的二次登记。等待中的领取绑定初始 channel generation，删除重建后返回 `ErrNoData`。

### 确认与恢复

ACK 原子删除 pending 数据及到期索引；重复 ACK 幂等。清理使用独立的有界 context，原业务 context 已取消也能确认。失败会记录日志并保留恢复凭据；调用方可再次调用 ack 函数。

回执登记两分钟后可恢复。每个 bus 每分钟唤起恢复循环，在十秒预算内按每批 100 条继续排空到期候选。也可显式调用 `Recover(ctx, limit)`，`limit` 为 1–1000；返回已确认重新入队数量及错误，部分成功与错误可同时返回。

单条恢复脚本重新检查是否仍未确认、是否到期，以及 generation 是否匹配。有效时先入队，再删除恢复记录，均在同一个脚本内；旧 generation 的回执直接清理，不重建旧队列。并发恢复者处理同一回执最多重新入队一次。

损坏记录保留数据和索引、报告错误，并延后五分钟重试，避免它永久占住全局到期队首。修复数据后可等待退避到期，不要清空整个 pending 索引。pending 数据不独立设置短 TTL，避免恢复时只剩索引、正文已消失。

## 可靠性边界

- 原子性指 Redis 端操作不会与其他命令交错，**不等于 Redis Lua 发生运行时错误后自动回滚**。脚本先校验类型、参数与记录结构，再执行修改。
- Redis 本身的数据持久性取决于宿主的 AOF/RDB、复制和故障切换配置；本库不提供 Redis 故障后的额外磁盘副本。
- 领取成功但响应丢失时，消息保留在 pending，超时后恢复；它不保证“恰好一次”。业务处理超过两分钟、ACK 失败、广播部分成功重试，都可能产生重复处理。
- 恢复消息回到队尾，不保证恢复后的严格顺序。
- ready 队列仍保留原有十分钟闲置 TTL；主动删除或最后订阅失活会删除该 channel 的积压，这是显式队列生命周期，不是持久任务事实源。
- 库内 `Subscriber.Handle` 保持原有 `defer ack()` 语义：业务回调返回错误也会 ACK。需要“业务失败不确认”的调用方应直接使用 `Pop` 并控制 ack 时机。
- 当前 v8/v9 adapter 接受普通 `*redis.Client`，多 key 脚本按单节点或兼容代理部署验证；**未实现 Redis Cluster 跨槽支持**。

## v3 → v4 切换步骤

1. 暂停所有旧版本消息生产入口，包括后台定时生产；保留旧消费者处理已有队列。
2. 确认旧 ready 队列和在途业务已排空。旧 ACK 有自动过期，不能仅凭 key 消失认定业务已经处理；还要核对业务事实源。必要时在维护窗口用限速、有限批次的 `SCAN` 做只读盘点，禁止 `KEYS`。
3. **如仍有必须保留的旧消息，停止发布流程。** 先制定并验证专门的数据迁移；新恢复器不会导入 v3 pending，不能直接删除旧 key。
4. 停止全部旧生产者／消费者进程，确认旧巡检器也退出。
5. 升级所有参与者，修改 manager 装配和空轮询错误处理；先启动并注册消费者，再恢复生产。
6. 验证注册、收发、ACK、超时恢复；检查扫描慢日志消失、授权连接等待与上报延迟是否下降。
7. 回滚同样先停写、排空 v4，再切回旧版。不能在 v4 尚有消息时直接切回 v3。

不要修改或删除 `session:event_id:*` 等业务去重数据来绕过这次修复。

## 本地验证

本机安装 `redis-server` 后执行：

```bash
go test -race ./xmsgbus ./xmsgbus/impl/memory ./xmsgbus/impl/redis/... -count=1
go test ./xmsgbus/... -run '^$'
```

集成测试为每个用例启动随机本机端口、禁用持久化的隔离 Redis，退出后关闭；不连接默认 `6379` 或线上实例。没有 `redis-server` 时这些用例会跳过，不能将跳过当作原子性验证通过。
