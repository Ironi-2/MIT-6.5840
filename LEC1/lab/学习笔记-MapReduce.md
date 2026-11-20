# Lab 1: MapReduce —— 手把手学习笔记（Windows 适配版）

## 0. 先建立一个大画面

想象你要统计一堆小说里每个单词出现了多少次。

**最笨的办法**：一个人从头读到尾，边读边记。——这就是 `mrsequential.go`（顺序版），慢。

**MapReduce 的办法**：雇一群人（worker），把小说分给每个人（map 任务），
每个人把自己读到的词记成小纸条（中间文件），
然后按字母分类：所有以 A 开头的词交给第 1 组，B 开头的交给第 2 组……（shuffle），
第 1 组的人把所有 A 开头的纸条汇总统计（reduce），最后拼起来。

这里有三个角色：
| 角色 | 现实比喻 | 代码 |
|------|----------|------|
| Coordinator | 包工头：派活、记账、发现有人掉线就换人 | `mr/coordinator.go` |
| Worker | 工人：领活、干活、交活 | `mr/worker.go` |
| 应用 (Map/Reduce 函数) | 具体怎么数词 | `mrapps/*.go`（本 Windows 适配版在 `main/mrworker_win.go`） |

---

## 第 1 步：先搞懂「顺序版」是怎么数词的

打开 `src/main/mrsequential.go`，它做三件事：
1. 读所有输入文件，对每个文件调用 `mapf`（即 wc.Map），得到一堆 `<词, "1">`；
2. 把所有 `<词, "1">` 按词排序（`sort.Sort(ByKey(...))`）——排序后相同的词挨在一起；
3. 从头扫一遍，遇到连续相同的词就把它们的 "1" 收集成列表，调用 `reducef`（即 wc.Reduce）数出个数，写成一行 `词 个数`。

> 这就是 MapReduce 的「灵魂」：**Map 产出键值对 → 按 key 分组 → Reduce 处理每组**。
> 分布式版本唯一的区别是：这些步骤分散到不同机器上做，中间多了「搬运数据」和「容错」。

---

## 第 2 步：worker 和 coordinator 怎么「说话」——RPC

两个进程之间通过网络调用函数，叫 RPC（远程过程调用）。你可以把它想象成**发短信**：

- worker 发短信问包工头：「有活吗？」（`AskForTask`）
- 包工头回：「有，去处理 `pg-grimm.txt`，这是第 3 个 map 任务，分 10 个桶。」（`AskForTaskReply`）
- worker 干完活发短信：「第 3 个 map 做完了。」（`TaskDone`）
- 做 map 的 worker 还要补一条：「我产出的 10 个中间文件在这些路径……」（`ReportMapFiles`）

**关键规则**（`src/mr/rpc.go`）：短信里带的字段**首字母必须大写**。
因为 Go 的 gob 编码只传「导出字段」，小写字段会被静默丢掉，对方收到的是零值。
这是做这个 lab 最常见的坑。

我们定义了三种任务类型：
```
TaskMap    = 0   // 去读一个输入文件，做 map
TaskReduce = 1   // 去拉中间数据，做 reduce
TaskWait   = 2   // 现在没活，睡会儿再来
TaskExit   = 3   // 全做完了，你可以下班了
```

---

## 第 3 步：Coordinator（包工头）的核心逻辑

打开 `src/mr/coordinator.go`。它手里有几本账：

```go
mapTasks    []*task   // 8 个 map 任务（每个输入文件一个）
reduceTasks []*task   // 10 个 reduce 任务
mapDone     int       // 已完成 map 数
reduceDone  int       // 已完成 reduce 数
mapFiles    [][]string // 每个 map 产出的 10 个中间文件路径
mu          sync.Mutex // 锁！
```

每个任务有三种状态：`idle`（没人领）→ `inProgress`（有人在做）→ `done`（做完了）。

### 为什么要加锁（`mu`）？
协调者是个**并发服务器**：多个 worker 可能同时来要活。
如果不加锁，两个 worker 可能同时看到「任务 3 是 idle」，都领走它——
这就出错了。加锁保证一次只有一个 worker 改账本。

### `AskForTask`（派活）的决策顺序
1. **有没有超时的任务？** 某个 worker 领走任务后 10 秒没消息，就认为它挂了，
   把任务重新标成 inProgress 派给别人（这就是容错）。
2. **有没有没领走的 map？** 有就派。
3. **map 全做完了吗？** 只有全做完了才派 reduce。
   > 这正是论文那道题的答案：reduce 必须等所有 map 完成，
   > 否则同一个 key 的 value 会散落在还没完成的文件里，漏统计。
4. 都没活但没做完 → 回 `TaskWait` 让 worker 睡 100ms 再来。
5. 全做完了 → 回 `TaskExit` 让 worker 退出。

### `ReportMapFiles`（记下中间文件在哪）
map worker 做完后调用它，把 10 个中间文件路径告诉协调者。
`gatherMapFiles(r)` 会把「所有 map 的第 r 个文件」收集起来，
这就是第 r 个 reduce 任务要读的全部数据（对应论文 Figure 1 第 5 步的 shuffle）。

### `Done()` 和 `allDone()`
所有 map 和 reduce 都 done 时返回 true，`main/mrcoordinator.go` 看到后就退出。

---

## 第 4 步：Worker（工人）的核心逻辑

打开 `src/mr/worker.go`。主体是一个**无限循环**：

```
for {
    问协调者要任务
    如果是 map    → doMap()
    如果是 reduce → doReduce()
    如果是 wait   → 睡 100ms
    如果是 exit   → 退出循环
}
```

### `doMap`（做 map 任务）分 5 步：
1. 读输入文件（`os.Open` + `ioutil.ReadAll`）
2. 调用用户的 Map 函数（比如 wc.Map），得到一堆 `<key, value>`
3. 对每个 key 算 `ihash(key) % nReduce`，决定它进哪个桶（0~9）
4. 每个桶写一个文件：`mr-<map号>-<桶号>`，内容用 **JSON** 编码
   （为什么用 JSON？因为 value 里可能有换行、空格，直接按文本存会解析错乱）
5. 汇报文件位置 + 销账（`ReportMapFiles` → `TaskDone`）

**⚠️ 重点：临时文件 + 原子 rename**
写文件时先写到 `mr-X-Y-tmp-xxxx` 临时文件，全部写完后 `os.Rename` 成最终名。
为什么？因为 lab 的 `TestCrashWorker` 会故意杀掉 worker。
如果直接写最终文件，崩溃时会留下半截内容，reduce 读到就错了。
`rename` 是原子操作：要么看到完整文件，要么看不到，绝不会看到半成品。

### `doReduce`（做 reduce 任务）分 4 步：
1. 从协调者给的路径列表里读所有中间文件（用 JSON 解码成 `<key,value>` 列表）
   > 在本 lab 里所有 worker 在同一台机器、共享文件系统，所以直接读文件即可。
   > 真实系统里这些文件在别的机器上，要靠 RPC 远程读（论文里就是这么做的）。
2. 按 key 排序 → 相同 key 的 value 挨在一起
3. 对每个 key 收集所有 value，调用用户的 Reduce 函数，
   按 `fmt.Fprintf("%v %v\n", key, output)` 格式写输出（**格式必须和顺序版一致**）
4. 同样用临时文件 + rename 写成 `mr-out-<reduce号>`，然后销账

---

## 第 5 步：容错是怎么工作的

**场景**：worker A 领了 map 任务 3，读了半个文件就崩了。
1. A 不再回应，协调者 10 秒后发现任务 3 超时；
2. 协调者把任务 3 重新标成可派发，派给 worker B；
3. B 从头做一遍，产出完整的中间文件；
4. 如果之前有 reduce worker 已经读了 A 的半截文件怎么办？——
   因为 A 写的是临时文件、崩溃时没 rename，所以最终文件名根本不存在，
   reduce 读不到它，会等 B 的完整文件。（这就是临时文件+rename 的第二个好处）

**场景**：协调者误以为 worker A 挂了，把同一个 map 任务派给了 A 和 B 两次。
- 两个 worker 都写 `mr-3-*` 文件，但都用原子 rename，最终文件是某一次完整的结果（不会混合）；
- 协调者对重复销账会忽略（`if t.state == taskDone { return nil }`）；
- 前提是 **Map/Reduce 是纯函数**（同样输入必得同样输出）——这是程序员的责任。

---

## 第 6 步：在本机怎么跑（Windows 适配版）

### 环境说明
MIT 原版用 `-buildmode=plugin` 动态加载应用（wc.so 等），
**Windows 版 Go 不支持 plugin**，所以官方 `make mr` 在 Windows 跑不了。
我们做了适配：`src/main/mrworker_win.go` 把各应用直接编译进 worker，
用第一个参数选择应用（`wc` / `indexer` / `crash` 等），核心 `mr/` 代码不变。

另外 Go 工具链装在 `LEC1/lab/_tools/go-install`（系统没装 Go），
`go.sh` 脚本帮你设好了环境变量。Unix socket 也改成了 TCP 端口。

### 编译
```bash
cd LEC1/lab
./go.sh build -o main/mrcoordinator main/mrcoordinator.go
./go.sh build -o main/mrworker_win main/mrworker_win.go
./go.sh build -o main/mrsequential_win main/mrsequential_win.go
```

### 跑单个测试
```bash
./runtest.sh wc 3      # 3 个 worker 跑词频统计
```

### 跑全套测试（7 项）
```bash
./runtests_all.sh
```
覆盖：wc/indexer 正确性、map 并行、reduce 并行、任务不重复、不提前退出、worker 崩溃容错。
✅ **本机实测：7/7 全部通过**（崩溃测试中有 5~8 次任务重派，输出仍与顺序版一致）。

### 在 Linux/WSL 上提交给 Gradescope 的话
核心代码就是 `src/mr/coordinator.go`、`src/mr/worker.go`、`src/mr/rpc.go` 这三个文件。
把它们拷到官方仓库的对应位置，`cd src && make mr` 跑官方测试即可。
（Windows 适配文件 `mrworker_win.go`、`mrsequential_win.go` 不需要提交。）

---

## 常见坑速查
1. RPC 字段没大写 → 对方收到零值，任务号永远是 0。
2. `reply` 传入前没初始化为零值 → gob 可能返回错误结果。
3. 忘了加锁 → 两个 worker 领到同一个任务（`TestJobCount` 会挂）。
4. 直接写最终文件不 rename → 崩溃时留下半截文件（`TestCrashWorker` 会挂）。
5. reduce 提前开始 → 漏数据（输出和顺序版不一致）。
6. 输出格式不对（多了/少了空格、换行）→ `runCmp` 逐字节比较会挂。
7. worker 提前退出 → `TestEarlyExit` 挂（我们的实现靠「连不上协调者就退出」避免）。
