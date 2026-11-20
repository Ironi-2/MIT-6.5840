# MIT 6.5840 — Distributed Systems（Spring 2026）

本课程学习材料与实验代码仓库。

## 关于课程

MIT 6.5840（原 6.824）是 MIT 计算机科学系的研究生核心课程，讲授**工程化分布式系统所需的抽象与实现技术**，三大主题为：

- **容错（Fault Tolerance）**
- **复制（Replication）**
- **一致性（Consistency）**

课程以经典论文研读 + 编程实验为主，主要使用 **Go** 语言。

- 课程主页：<https://pdos.csail.mit.edu/6.5840/>
- 课程日程：<https://pdos.csail.mit.edu/6.5840/schedule.html>

## 仓库结构

仓库按讲次组织，共 22 个文件夹 `LEC1/` ～ `LEC22/`，每个讲次文件夹内有一个 `lab/` 子目录用于存放该讲的实验代码。

```
MIT-6.5840/
├── README.md           # 本文件
├── CONTEXT.md          # 项目内文件操作约定（讲次—主题对照表等）
├── LEC1/
│   ├── mapreduce.pdf
│   └── lab/            # Lab 1: MapReduce 实验代码与笔记
├── LEC2/lab/
├── ...
└── LEC22/lab/
```

更完整的讲次—主题对照、实验归属与文件操作约定见 [CONTEXT.md](CONTEXT.md)。

## 实验（Labs）

| 实验 | 内容 | 对应讲次 |
|---|---|---|
| Lab 1 | MapReduce | LEC1 |
| Lab 2 | Key/Value server | LEC2 |
| Lab 3 | Raft（3A/3B/3C/3D） | LEC6 / LEC7 |
| Lab 4 | 基于 Raft 的 KV 服务器 | LEC9 / LEC10 |
| Lab 5 | 分片 KV 系统（Sharded KV） | LEC14 |

实验框架来自 MIT 官方实验仓库，代码与学习笔记放在对应的 `LEC<n>/lab/` 下。编译产物、运行时临时目录和日志已通过 `.gitignore` 排除，仓库只保留源码、脚本与笔记。

## 学习路线建议

1. 按 [课程日程](https://pdos.csail.mit.edu/6.5840/schedule.html) 的 Preparation 列表逐讲阅读经典论文（MapReduce、GFS、Paxos、Raft、Spanner 等）。
2. 实验按 Lab 1 → 5 顺序完成，代码归档到对应讲次的 `lab/` 目录。
3. 需要定位某一讲时，参照 `CONTEXT.md` 的讲次—主题对照表。
