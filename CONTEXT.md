# 项目上下文：MIT 6.5840 分布式系统（Spring 2026）

本文件是项目级说明，供每个新对话开始时读取。请遵循以下约定。

## 项目概况

- 课程：MIT 6.5840 Distributed Systems，Spring 2026
- 会话当前工作目录：**本文件所在的项目根目录**（`MIT 6.5840/`）
- 实验主要使用 **Go** 语言

## 目录结构

根目录下按讲次建有 22 个文件夹，命名格式为 `LEC<数字>`（无前导零）：

```
MIT 6.5840/
├── CONTEXT.md            ← 本说明文件
├── LEC1/
│   ├── mapreduce.pdf
│   └── lab/              ← 该讲的实验代码目录
├── LEC2/lab/
├── ...
└── LEC22/lab/
```

## 操作规则（重要）

1. **按讲次定位目录**：当用户提到某一讲时，所有文件操作都应在对应的 `LEC<n>/` 文件夹下进行；实验代码一律放入该讲的 `LEC<n>/lab/` 子目录。
2. **讲次识别**：用户可能用编号（如「第 6 讲」「LEC6」）或主题（如「Raft 那讲」「MapReduce 那讲」）指代。参照下方「讲次—主题对照表」确定编号。
3. **相对路径**：由于工作目录是根目录，路径统一写成 `LEC<n>/...` 或 `LEC<n>/lab/...`，例如 `LEC6/lab/raft.go`。
4. **不确定时先问**：若无法从用户描述判断是哪一讲，先询问确认，不要猜。

## 讲次—主题对照表

| 讲次 | 主题 |
|---|---|
| LEC1 | Introduction / MapReduce（实验：Lab 1） |
| LEC2 | RPC and Threads |
| LEC3 | GFS |
| LEC4 | Paxos |
| LEC5 | Go patterns（客座：Russ Cox） |
| LEC6 | Fault Tolerance: Raft (1) |
| LEC7 | Fault Tolerance: Raft (2) |
| LEC8 | Consistency and Linearizability |
| LEC9 | ZooKeeper |
| LEC10 | Q&A Lab 3A+B |
| LEC11 | Distributed Transactions（2PC） |
| LEC12 | Spanner |
| LEC13 | Chain Replication |
| LEC14 | Optimistic Concurrency Control（FaRM） |
| LEC15 | Verification of distributed systems（IronFleet） |
| LEC16 | Cache Consistency: Memcached at Facebook |
| LEC17 | AWS Lambda |
| LEC18 | Ray |
| LEC19 | Fork Consistency, SUNDR |
| LEC20 | Peer-to-peer: Bitcoin |
| LEC21 | Byzantine Fault Tolerance（PBFT） |
| LEC22 | Project demos |

## 实验—讲次提示

- Lab 1（MapReduce）→ LEC1
- Lab 2（Key/Value server）→ LEC2
- Lab 3（Raft）→ LEC6 / LEC7
- Lab 4（KV Raft）→ LEC9 / LEC10
- Lab 5（Sharded KV）→ LEC14

## 说明

本文件只提供约定与指引，不会自动改变会话的工作目录；它依赖 agent 在对话开始时读取并遵守这些规则。若某次对话中 agent 未按规则定位目录，可直接提醒它「先读根目录的 CONTEXT.md」。
