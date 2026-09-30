# 集群架构设计：一主多从 (Master-Node)

`sing-controller` 支持分布式集群管理架构，实现集中面板管控跨地域、NAT 穿透的多台 `sing-box` 代理节点。

---

## 1. 总体拓扑

```mermaid
graph TD
    User([管理员 / WebUI]) -->|HTTPS / REST API| Master[Master 主节点<br/>Cluster Hub WS :25774]
    
    subgraph Nodes [分布式从节点 (NAT 友好)]
        NodeA[从节点 A (NAT 内网)<br/>Agent -> 出站 WSS]
        NodeB[从节点 B (公网 VPS)<br/>Agent -> 出站 WSS]
        NodeC[从节点 C<br/>Agent -> 出站 WSS]
    end

    Master -.->|WSS 任务下发 & 心跳| NodeA
    Master -.->|WSS 任务下发 & 心跳| NodeB
    Master -.->|WSS 任务下发 & 心跳| NodeC

    NodeA --- BoxA[本地 sing-box 实例]
    NodeB --- BoxB[本地 sing-box 实例]
    NodeC --- BoxC[本地 sing-box 实例]
```

### 核心特性
- **控制面 / 数据面分离**：Master 仅负责调度任务与配置下发，流量在各个 sing-box 节点间直连，Master 永不中转流量。
- **从节点主动出站**：基于 WebSocket/WSS 主动连接 Master，从节点位于 NAT/内网后无需公网 IP 和任何入站映射端口。
- **单二进制自适应**：同一个 `sing-controller` 二进制，通过 `cluster.mode` 决定是作为 `master` 还是 `node` 运行。
- **原子安全校验管线**：Master 下发配置至从节点后，从节点本地执行 `box.New` 试跑校验通过后才落盘，非法配置拒绝并上报错误。

---

## 2. 配置方式

### Master 主节点增量配置 (`controller.json`)
```json
{
  "cluster": {
    "mode": "master",
    "hub_listen": "0.0.0.0:25774",
    "token_file": "./cluster-tokens.json"
  }
}
```

### Node 从节点增量配置 (`controller.json`)
```json
{
  "cluster": {
    "mode": "node",
    "master_url": "wss://panel.example.com:25774/cluster/v2/rpc",
    "token": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "node_id": "hk-01",
    "heartbeat_interval": "5s"
  }
}
```
