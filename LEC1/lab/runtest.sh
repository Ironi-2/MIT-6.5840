#!/usr/bin/env bash
# Windows 下的 Lab1 端到端测试脚本（模拟官方 mr_test.go 的 TestWc）
# 用法： ./runtest.sh wc 3
set -u
LAB="E:/项目/并行与分布式系统/MIT 6.5840/LEC1/lab"
SRC="$LAB/src"
APP="${1:-wc}"
NWORKERS="${2:-3}"
SOCK=":29000"  # Windows 下用 TCP 端口
TMPDIR="$SRC/mr-tmp-test"

# 准备一个干净的工作目录（worker 在里面读写中间文件）
rm -rf "$TMPDIR" && mkdir -p "$TMPDIR"
cp "$SRC"/main/pg-*.txt "$TMPDIR"/
cp "$SRC/main/mrcoordinator" "$SRC/main/mrworker_win" "$SRC/main/mrsequential_win" "$TMPDIR"/

cd "$TMPDIR" || exit 1

# 1) 先跑顺序版，生成「标准答案」
./mrsequential_win pg-*.txt >/dev/null 2>&1 || { echo "SEQ FAILED"; exit 1; }
sort mr-out-0 > correct.txt
rm -f mr-out-0

# 2) 启动协调者
./mrcoordinator "$SOCK" pg-*.txt >coord.log 2>&1 &
COORD_PID=$!
sleep 1

# 3) 启动 N 个 worker
for i in $(seq 1 "$NWORKERS"); do
  ./mrworker_win "$APP" "$SOCK" >"worker-$i.log" 2>&1 &
done

# 4) 等协调者结束（最多 120 秒）
wait "$COORD_PID"
CSTATUS=$?
sleep 1

# 5) 合并输出并对比
cat mr-out-* 2>/dev/null | sort > all.txt
echo "===== coordinator exit: $CSTATUS ====="
echo "===== output files: $(ls mr-out-* 2>/dev/null | wc -l) ====="
if cmp -s all.txt correct.txt; then
  echo "===== PASS: 分布式输出 ==  顺序版标准答案 ====="
else
  echo "===== FAIL: 输出不一致 ====="
  echo "--- diff (前 30 行) ---"
  diff all.txt correct.txt | head -30
fi
echo "===== coordinator log (tail) ====="
tail -8 coord.log
echo "===== workers log (tail) ====="
tail -3 worker-*.log 2>/dev/null | head -30
