#!/usr/bin/env bash
# 单独跑崩溃容错测试（带监护人，模拟官方 TestCrashWorker）
set -u
LAB="E:/项目/并行与分布式系统/MIT 6.5840/LEC1/lab"
D="$LAB/src/win-tests/cw"
cd "$D" || exit 1
rm -f mr-out-* mr-*-tmp-* w*.log coord.log
PORT=29216

# 标准答案（用不崩溃的 nocrash 应用生成）
./mrsequential_win nocrash pg-*.txt >/dev/null 2>&1
sort mr-out-0 > correct.txt; rm -f mr-out-0

./mrcoordinator ":$PORT" pg-*.txt >coord.log 2>&1 & CPID=$!
sleep 1
for i in 1 2 3; do
  ( while true; do
      ./mrworker_win crash ":$PORT" >>"w$i.log" 2>&1
      RC=$?
      [ "$RC" -eq 0 ] && break
      sleep 0.2
    done ) &
done
wait "$CPID"; CS=$?
sleep 1
cat mr-out-* 2>/dev/null | sort > all.txt
echo "coordinator exit=$CS"
echo "restarts=$(grep -c 're-starting' coord.log)"
if cmp -s all.txt correct.txt; then echo "PASS: 崩溃容错输出正确"; else echo "FAIL:"; diff all.txt correct.txt | head -10; fi
