# Gnode

Gnode 是一个基于 Go 实现的轻量级区块链节点系统，支持钱包管理、交易签名验签、UTXO 账本、区块持久化、gRPC P2P 通信以及基础链同步能力。

## 项目简介

本项目基于 Go 实现一个支持多节点通信与链同步的轻量级区块链节点系统。项目采用分层架构设计，划分为 CLI 层、节点与网络层、服务层、核心领域层和基础设施层，各层职责边界清晰，便于功能迭代、问题排查和后续扩展。

核心领域层基于区块 Hash 链维护账本顺序，通过 UTXO 模型实现余额计算、交易输入输出管理与双花校验；基础设施层基于 bbolt 嵌入式 KV 数据库存储区块、链状态、UTXO 集合及钱包数据，并通过原子化数据库更新保证区块索引、最佳链状态和 UTXO 状态的一致性。

交易模块基于 ECDSA 实现签名与验签，保证交易发起方权属与数据完整性。网络层基于 gRPC 构建 P2P 节点通信能力，支持 Peer 配置、Ping 探活、链状态查询、按高度拉取区块及基础链同步。系统同时提供 CLI 操作入口与 JSON 配置管理能力，支持多节点联调、独立数据库运行和节点长期运行。

## 功能特性

- 支持钱包创建、查询和持久化
- 支持 ECDSA 交易签名与验签
- 基于 UTXO 模型实现余额计算、输入输出管理和双花校验
- 基于区块 Hash 链维护账本顺序
- 支持基础 Proof-of-Work 校验
- 使用 bbolt 持久化区块、链状态、UTXO 集合和钱包数据
- 提供 CLI 命令完成初始化、转账、余额查询、链信息打印和节点启动
- 使用 JSON 配置管理节点 ID、监听地址、数据库路径和 Peer 列表
- 基于 gRPC 实现 P2P 节点通信
- 支持 Peer Ping、链状态查询、按高度拉取区块和基础链同步
- 使用确定性交易哈希编码，避免 gob 元数据导致的跨节点哈希不一致问题

## 技术栈

- Go
- gRPC
- Protocol Buffers
- bbolt
- ECDSA
- UTXO 模型
- Hash 链
- CLI

## 项目结构

```text
.
├── cmd/                    # CLI 入口和命令解析
├── configs/                # 示例节点配置和 CLI 配置
├── docs/                   # 架构说明文档
├── internal/
│   ├── config/             # JSON 配置加载和默认配置管理
│   ├── core/               # 核心领域层：区块、交易、UTXO、链状态
│   ├── infra/database/     # 数据库抽象和 bbolt 实现
│   ├── node/               # 长期运行节点、Peer 管理和同步调度
│   ├── p2p/                # gRPC Server/Client 网络传输层
│   ├── service/            # 应用服务层
│   └── wallet/             # 钱包模型和钱包存储
└── pkg/                    # 通用加密和工具包
```

## 配置说明

节点配置示例：

```json
{
  "version": 1,
  "node_id": "node1",
  "listen_addr": "127.0.0.1:5001",
  "chain_db": "cmd/blockchain.db",
  "wallet_db": "cmd/wallet.db",
  "peers": ["127.0.0.1:5002"]
}
```

字段说明：

- `node_id`：本地节点 ID
- `listen_addr`：gRPC 监听地址
- `chain_db`：区块链数据库文件路径
- `wallet_db`：钱包数据库文件路径
- `peers`：当前节点需要连接的 Peer 地址列表

CLI 当前默认节点配置路径保存在 `configs/cli_config.json` 中。

## 快速开始

### 1. 编译

```bash
go build -o gnode ./cmd
```

也可以直接使用 `go run` 执行命令：

```bash
go run ./cmd <command> [args]
```

### 2. 设置默认节点配置

```bash
./gnode config use ./configs/default_node.json
```

查看当前默认配置：

```bash
./gnode config show
```

### 3. 初始化本地链

```bash
./gnode init ./configs/default_node.json
```

如果没有传入矿工地址，程序会自动创建或读取 `worker` 矿工钱包，并使用该钱包生成创世区块。

### 4. 创建钱包

```bash
./gnode create-wallet alice
```

查看钱包信息：

```bash
./gnode get-wallet alice
```

### 5. 发起转账

使用 `get-wallet` 输出的钱包地址作为收款地址：

```bash
./gnode transfer worker <alice-address> 20
```

打印当前区块链：

```bash
./gnode print
```

## 两节点本地运行示例

仓库中提供了两个示例配置：

- `configs/default_node.json`：监听 `127.0.0.1:5001`
- `configs/default_node2.json`：监听 `127.0.0.1:5002`

典型运行流程：

```bash
# 终端 1：初始化 node1，并产生一些链上数据
./gnode config use ./configs/default_node.json
./gnode init ./configs/default_node.json
./gnode create-wallet alice
./gnode get-wallet alice
./gnode transfer worker <alice-address> 20

# 终端 2：初始化 node2，用于从 Peer 同步
./gnode sync-init ./configs/default_node2.json

# 终端 1：启动 node1
./gnode node ./configs/default_node.json

# 终端 2：启动 node2
./gnode node ./configs/default_node2.json
```

当 node2 检测到 node1 的链高度更高时，会按高度向 node1 拉取缺失区块，在本地完成区块校验、交易验签、Proof-of-Work 校验、落库和 UTXO 更新。

## CLI 命令

```text
init [configFilePath] [minerAddress]       初始化链并创建创世区块
sync-init [configFilePath] [minerAddress]  初始化用于同步的本地链存储
config show                                查看当前默认节点配置
config use <configFilePath>                设置默认节点配置
config reset                               重置默认节点配置
create-wallet <username> [role]            创建钱包，默认 role 为 user
get-wallet <username>                      查看钱包信息
list-wallets                               列出所有钱包
balance <username>                         查询钱包余额
transfer <fromUser> <toAddress> <amount>   发起转账并打包新区块
print                                      打印当前区块链
reset-chain [--with-wallets]               删除本地链数据库，可选删除钱包数据库
node [configFilePath]                      启动长期运行节点
```

## P2P RPC 接口

接口定义位于 `internal/p2p/proto/peer.proto`。

当前支持：

- `Ping`：节点探活
- `GetChainState`：查询 Peer 的最佳高度和最佳区块 Hash
- `GetBlocksFromHeight`：按高度拉取 Peer 的区块数据

## 数据持久化

Gnode 使用 bbolt 作为嵌入式 KV 数据库。当前主要存储内容包括：

- 区块数据
- 区块高度索引
- 最佳链状态
- UTXO 集合
- 钱包数据

链数据库和钱包数据库使用独立文件，便于多个节点使用不同本地状态进行联调。



