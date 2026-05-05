# EasyWired

## 项目解决的事情

EasyWired 是一个运行在各节点上的 WireGuard 控制面 Agent，用来自动完成节点接入、Peer 信息交换、WireGuard 内网 IP 分配、Peer 查询、断开连接、配置持久化和 WireGuard 配置生成/应用。

它不修改 WireGuard 协议，也不实现新的 VPN 协议；数据面仍由官方 WireGuard 负责，EasyWired 只负责控制面编排。

适合的场景：先让新节点连接一个可达节点，拿到内网地址和 Peer 配置；之后节点之间可以通过 WireGuard 内网地址继续互相发现和建立 Peer。

## 项目使用方法

构建：

```powershell
go build ./cmd
```

启动一个可分配地址的节点：

```powershell
.\cmd.exe serve --config examples/nodeA.json --device wg0 --listen :8080 --backend auto --output nodeA-wg0.conf
```
新节点连接到 nodeA：

```powershell
.\cmd.exe connect --config examples/nodeB.json --device wg0 --url http://nodeA:8080/connect --backend auto --output nodeB-wg0.conf
```

查询远端 Peer：

```powershell
.\cmd.exe peers --url http://nodeA:8080/peers
```

断开 Peer：

```powershell
.\cmd.exe disconnect --config examples/nodeB.json --device wg0 --url http://nodeA:8080/disconnect --public-key <peer-public-key> --backend auto --output nodeB-wg0.conf
```

只导出 WireGuard 配置：

```powershell
.\cmd.exe export --config examples/nodeB.json --output nodeB-wg0.conf
```
Backend 选择：

- `auto`：Linux 默认 `linux-native`，其他平台默认 `wgconfig-file`
- `wgconfig-file`：只生成标准 WireGuard `.conf`
- `linux-native`：Linux 下通过 `wgctrl` 和 `netlink` 应用配置
- `windows-service` / `windows-nt`：Windows native 接口已预留，当前 native apply 返回未实现

## 项目的技术原理

### 整个 Agent 的流程

每个节点都运行一个 Agent，并维护本地 JSON 状态：

```text
NodeConfig = NodeID + Interface + Peers + ExtField + Leases
```

- `Interface`：本节点 WireGuard 接口信息
- `Peers`：本节点已经认识的 WireGuard Peer
- `ExtField`：推荐给对方连接本节点时使用的元数据
- `Leases`：Join 模式下已经分配出去的内网 IP

```mermaid
flowchart TD
    CLI[CLI 命令] --> Agent[EasyWired Agent]
    HTTP[HTTP API] --> Agent

    Agent --> Store[JSON Store]
    Agent --> IPAM[IPAM 地址分配]
    Agent --> Backend[Backend 抽象层]

    Store --> Config[(NodeConfig)]
    IPAM --> Leases[(Leases)]

    Backend --> Linux[linux-native\nwgctrl + netlink]
    Backend --> File[wgconfig-file\n导出 wg-quick .conf]
    Backend --> Windows[windows-service / windows-nt\n接口预留]

    Linux --> WG[WireGuard 设备]
    File --> Conf[WireGuard 配置文件]
```

Agent 的核心思想是：控制面负责“发现、交换、分配、持久化”，数据面仍然交给 WireGuard。

```text
控制面 = HTTP API + Store + IPAM + Backend 编排
数据面 = WireGuard 原生隧道
```

### Join 模式示例

假设 `nodeA` 已经有地址 `10.0.0.1/24`，`nodeB` 只有公钥，还没有 WireGuard 内网地址。

```mermaid
sequenceDiagram
    participant B as nodeB
    participant A as nodeA
    participant S as nodeA JSON Store
    participant W as WireGuard Backend

    B->>A: POST /connect\npublicKey=bbb,address为空
    A->>A: 判断 Ready
    A->>A: 从 10.0.0.1/24 分配 10.0.0.2/32
    A->>S: 保存 lease 和 peer(nodeB)
    A->>W: 应用或导出 WireGuard 配置
    A-->>B: 返回 assigned=10.0.0.2/32\n返回 nodeA 的 Peer 配置
    B->>B: 保存 assigned address 和 nodeA peer
    B->>W: 应用或导出 WireGuard 配置
```

IP 分配公式可以理解为：

```text
available_ips = cidr_hosts(interface.address)
              - {network, broadcast, self_ip}
              - peer_allowed_ips
              - leased_ips

assigned_ip = min(available_ips)
```

所以当 `nodeA = 10.0.0.1/24` 时，第一次 Join 通常得到：

```text
nodeB => 10.0.0.2/32
nodeC => 10.0.0.3/32
```

### Peer 模式示例

当 `nodeB` 和 `nodeC` 都已经通过 `nodeA` 获得地址后，它们可以通过 WireGuard 内网互相连接。

```mermaid
sequenceDiagram
    participant B as nodeB\n10.0.0.2/32
    participant C as nodeC\n10.0.0.3/32

    B->>C: POST /connect\npublicKey=bbb,address=10.0.0.2/32
    C->>C: 判断请求方已有 address，进入 Peer 模式
    C->>C: 不分配新 IP
    C->>C: 保存 peer(nodeB)，AllowedIPs=10.0.0.2/32
    C-->>B: 返回 nodeC peer，AllowedIPs=10.0.0.3/32
    B->>B: 保存 peer(nodeC)
```

Peer 模式的路由规则固定为：

```text
AllowedIPs(peer-mode) = peer.address/32
```

这样不会把 `0.0.0.0/0` 或大网段路由下发给普通 Peer，避免多个 Peer 之间出现 WireGuard 路由冲突。

### Backend 抽象

```mermaid
classDiagram
    class Backend {
        +Name() string
        +Platform() string
        +EnsureDevice(ctx, device, cfg) error
        +ApplyInterface(ctx, device, iface) error
        +AddOrUpdatePeer(ctx, device, peer) error
        +RemovePeer(ctx, device, publicKey) error
        +ApplyConfig(ctx, device, cfg) error
        +ExportConfig(ctx, cfg) []byte
        +SupportsNativeApply() bool
    }

    Backend <|.. LinuxNativeBackend
    Backend <|.. WGConfigFileBackend
    Backend <|.. WindowsServiceBackend
    Backend <|.. WindowsNTBackend
```

Linux 下可以通过 `linux-native` 直接配置 WireGuard；其他平台初版默认使用 `wgconfig-file` 生成标准 `.conf`，再交给官方 WireGuard 客户端导入。Windows native 接口已预留，但当前不直接调用系统 WireGuard 服务。

因此跨平台控制面逻辑保持一致，平台差异集中在 backend 中。
