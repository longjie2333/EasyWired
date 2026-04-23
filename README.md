# EasyWired

一个为方便 WireGuard 节点之间配置分发的工具。通过提供中心服务，让节点启动后自动注册，服务中心统一管理节点并分发连接信息，帮助节点之间快速建立加密通信，省去了手动配置和分发的过程。

## 功能特点
- 自动节点注册：节点启动后自动上报 `nodeId`、公钥、地址等信息。
- 统一节点清单：可通过接口或命令查看已注册节点。
- 节点互联编排：服务中心可把两个节点配对并生成双方 Peer 信息。
- 动态配置下发：节点通过 SSE 接收 `peer-added` 事件并热更新 WireGuard Peer。
- 地址自动分配：连接时按目标节点网段自动分配可用 IPv4 地址。

## 功能概览
```mermaid
graph LR;
  A["节点A"] -->|1. 报到注册| S["服务中心"];
  B["节点B"] -->|1. 报到注册| S;
  S -->|2. 下发连接信息| A;
  S -->|2. 下发连接信息| B;
  A <-->|3. 建立加密通信| B;
```

## 接口流程
```mermaid
sequenceDiagram
    autonumber
    participant NA as 节点A(Client)
    participant NB as 节点B(Client)
    participant SC as 服务中心(Server)

    NA->>SC: POST /nodes/register (nodeId, publicKey, endpoint...)
    NB->>SC: POST /nodes/register (...)
    NA->>SC: GET /nodes/{id}/events (SSE 订阅)
    NB->>SC: GET /nodes/{id}/events (SSE 订阅)
    NA->>SC: GET /nodes/{id}/config
    NB->>SC: GET /nodes/{id}/config
    Note over NA,NB: 本地调用 wg.ApplyConfig 应用初始配置

    NA->>SC: POST /connect (fromNodeId=NA,toNodeId=NB)
    SC->>SC: 校验节点/分配可用IP/生成双方 Peer
    SC-->>NA: SSE event: peer-added
    SC-->>NB: SSE event: peer-added
    Note over NA,NB: 本地调用 wg.ApplyPeerHotUpdate 热更新 Peer
```
