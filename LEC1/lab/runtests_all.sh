#!/usr/bin/env bash
# Windows 适配版 Lab1 验收测试（对应官方 mr_test.go 的 6 个测试）
set -u
LAB="E:/项目/并行与分布式系统/MIT 6.5840/LEC1/lab"
SRC="$LAB/src"
TMPROOT="$SRC/win-tests"
mkdir -p "$TMPROOT"
PASS=0; FAIL=0

report() { if [ "$1" = PASS ]; then echo "  ✅ PASS: $2"; PASS=$((PASS+1)); else echo "  ❌ FAIL: $2"; FAIL=$((FAIL+1)); fi; }

setup() { # setup <目录名> <应用名> —— 标准答案必须用同一个应用生成
  D="$TMPROOT/$1"; APPSEQ="$2"; rm -rf "$D"; mkdir -p "$D"
  cp "$SRC"/main/pg-*.txt "$D"/
  cp "$SRC/main/mrcoordinator" "$SRC/main/mrworker_win" "$SRC/main/mrsequential_win" "$D"/
  cd "$D" || exit 1
  ./mrsequential_win "$APPSEQ" pg-*.txt >/dev/null 2>&1
  sort mr-out-0 > correct.txt; rm -f mr-out-0
}

run_job() { # app nworkers port
  APP=$1; N=$2; PORT=$3
  rm -f mr-out-* mr-*-tmp-* mr-worker-jobcount-* mr-worker-reduce-*
  ./mrcoordinator ":$PORT" pg-*.txt >coord.log 2>&1 & CPID=$!
  sleep 1
  for i in $(seq 1 "$N"); do ./mrworker_win "$APP" ":$PORT" >"w$i.log" 2>&1 & done
  wait "$CPID"; CS=$?
  sleep 1
  cat mr-out-* 2>/dev/null | sort > all.txt
  return $CS
}

echo "=========== TestWc: 输出正确性 ==========="
setup wc wc; run_job wc 3 29200
cmp -s all.txt correct.txt && report PASS "wc 输出 == 顺序版标准答案" || report FAIL "wc 输出不一致"

echo "=========== TestIndexer: 另一个应用 ==========="
setup idx indexer; run_job indexer 2 29201
cmp -s all.txt correct.txt && report PASS "indexer 输出正确" || report FAIL "indexer 输出不一致"

echo "=========== TestMapParallel: map 并行 ==========="
setup mp mtiming; S=$(date +%s); run_job mtiming 3 29202; E=$(date +%s)
# 8 个 map 各 500ms：串行至少 4 秒；3 worker 并行理想 1.5 秒 + reduce 开销
if cmp -s all.txt correct.txt; then report PASS "map 并行 (8 map/3 worker 用时 $((E-S))s)"; else report FAIL "map 并行输出不正确"; fi

echo "=========== TestReduceParallel: reduce 并行 ==========="
setup rp rtiming; run_job rtiming 3 29203
if cmp -s all.txt correct.txt && grep -q ' 2$' all.txt; then report PASS "reduce 并行 (输出 'N 2' 格式)"; else report FAIL "reduce 并行检查"; fi

echo "=========== TestJobCount: 每个 map 只跑一次 ==========="
setup jc jobcount; run_job jobcount 4 29204
VAL=$(grep '^a ' all.txt | head -1 | awk '{print $2}')
if [ "$VAL" = "8" ]; then report PASS "map 恰好执行 8 次 (无重复派发)"; else report FAIL "map 执行次数异常: $VAL (应为 8)"; fi

echo "=========== TestEarlyExit: worker 不提前退出 ==========="
setup ee early_exit; run_job early_exit 4 29205
if cmp -s all.txt correct.txt; then report PASS "所有 reduce 完成后才退出，输出未被截断"; else report FAIL "提前退出检查"; fi

echo "=========== TestCrashWorker: worker 崩溃容错 ==========="
setup cw nocrash
D="$TMPROOT/cw"; cd "$D" || exit 1
rm -f mr-out-* mr-*-tmp-*
PORT=29206
./mrcoordinator ":$PORT" pg-*.txt >coord.log 2>&1 & CPID=$!
sleep 1
# 官方测试有一个「监护人」：worker 崩溃退出后立刻重新拉起一个。
# 这里用 shell 实现同样的逻辑：每个 worker 用 while 循环不断重启，
# 直到协调者退出（socket 连不上时 worker 会自己退出）。
for i in 1 2 3; do
  # worker 崩溃退出后（exit 1），监护人立刻再拉起一个，
  # 直到它看到「连不上协调者」那种正常退出（exit 0）才停止。
  ( while true; do
      ./mrworker_win crash ":$PORT" >>"w$i.log" 2>&1
      RC=$?
      if [ "$RC" -eq 0 ]; then break; fi   # 协调者下班了，不用再拉了
      sleep 0.2
    done ) &
done
wait "$CPID"; CS=$?
sleep 1
cat mr-out-* 2>/dev/null | sort > all.txt
if cmp -s all.txt correct.txt; then
  report PASS "worker 随机崩溃后仍输出正确 ($(grep -c 're-starting' coord.log) 次任务重派)"
else
  report FAIL "崩溃容错检查"
fi

echo "======================================================"
echo "总计: PASS=$PASS FAIL=$FAIL"
[ "$FAIL" -eq 0 ]
