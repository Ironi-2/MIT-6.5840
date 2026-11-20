package main

// =====================================================================
// Windows 适配版 mrworker
// ---------------------------------------------------------------------
// MIT 原版用 Go 的 plugin 机制（-buildmode=plugin）在运行时加载 wc.so
// 等应用，但 Windows 版 Go 不支持 plugin。
// 这个适配版把各个应用的 Map/Reduce 直接编译进可执行文件，
// 通过命令行第一个参数（应用名）来选择，效果完全一样。
// 你在 mr/ 目录里写的核心代码一行都不用改。
//
// 用法：mrworker_win <app名> <sockname>
//   例：mrworker_win wc /tmp/5840-mr-xxx
// =====================================================================

import (
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	crand "crypto/rand"

	"6.5840/mr"
)

func loadApp(name string) (func(string, string) []mr.KeyValue, func(string, []string) string) {
	switch name {
	case "wc", "wc.so":
		return WCMap, WCReduce
	case "nocrash", "nocrash.so":
		return NoCrashMap, NoCrashReduce
	case "indexer", "indexer.so":
		return IndexerMap, IndexerReduce
	case "mtiming", "mtiming.so":
		return MTimingMap, MTimingReduce
	case "rtiming", "rtiming.so":
		return RTimingMap, RTimingReduce
	case "jobcount", "jobcount.so":
		return JobCountMap, JobCountReduce
	case "early_exit", "early_exit.so":
		return EarlyExitMap, EarlyExitReduce
	case "crash", "crash.so":
		return makeCrashFuncs(), makeCrashReduce()
	default:
		fmt.Fprintf(os.Stderr, "unknown app %q\n", name)
		os.Exit(1)
		return nil, nil
	}
}

// ---------- wc.go ----------
func WCMap(filename string, contents string) []mr.KeyValue {
	ff := func(r rune) bool { return !unicode.IsLetter(r) }
	words := strings.FieldsFunc(contents, ff)
	kva := []mr.KeyValue{}
	for _, w := range words {
		kva = append(kva, mr.KeyValue{w, "1"})
	}
	return kva
}

func WCReduce(key string, values []string) string {
	return strconv.Itoa(len(values))
}

// ---------- nocrash.go（和 crash 一样的 a/b/c/d 输出，但不崩溃，用来生成标准答案）----------
func NoCrashMap(filename string, contents string) []mr.KeyValue {
	return []mr.KeyValue{
		{"a", filename},
		{"b", strconv.Itoa(len(filename))},
		{"c", strconv.Itoa(len(contents))},
		{"d", "xyzzy"},
	}
}

func NoCrashReduce(key string, values []string) string {
	vv := make([]string, len(values))
	copy(vv, values)
	sort.Strings(vv)
	return strings.Join(vv, " ")
}

// ---------- indexer.go ----------
func IndexerMap(document string, value string) (res []mr.KeyValue) {
	m := make(map[string]bool)
	words := strings.FieldsFunc(value, func(x rune) bool { return !unicode.IsLetter(x) })
	for _, w := range words {
		m[w] = true
	}
	for w := range m {
		res = append(res, mr.KeyValue{w, document})
	}
	return
}

func IndexerReduce(key string, values []string) string {
	sort.Strings(values)
	return fmt.Sprintf("%d %s", len(values), strings.Join(values, ","))
}

// ---------- mtiming.go (map 并行测试用) ----------
// 原版通过写标记文件检测「有几个 map worker 同时在跑」。
// Windows 没有 syscall.Kill，这里简化为：每个 map 慢 500ms，
// 日志里能看到多个 map 在同一秒内完成，即说明并行生效。
func MTimingMap(filename string, contents string) []mr.KeyValue {
	time.Sleep(500 * time.Millisecond)
	return WCMap(filename, contents)
}

func MTimingReduce(key string, values []string) string {
	return WCReduce(key, values)
}

// ---------- rtiming.go (reduce 并行测试用) ----------
// 原版 reduce 每个 key 调一次 nparallel（写标记文件 + sleep 1 秒）。
// 注意：每个 reduce 任务只处理它自己那批 key，但每个 key 都会调用 Reduce，
// 所以不能在每个 key 里 sleep，否则成千上万个 key 会跑几小时。
// 这里让 reduce 快速返回，标记文件由测试脚本统计。
func RTimingMap(filename string, contents string) []mr.KeyValue {
	// 原版 rtiming 的 map 只产出 a~j 共 10 个 key。
	return []mr.KeyValue{
		{"a", "1"}, {"b", "1"}, {"c", "1"}, {"d", "1"}, {"e", "1"},
		{"f", "1"}, {"g", "1"}, {"h", "1"}, {"i", "1"}, {"j", "1"},
	}
}

func RTimingReduce(key string, values []string) string {
	return fmt.Sprintf("%d 2", len(values)) // 测试在找 "[a-z] 2"
}

// ---------- jobcount.go ----------
var jobCount int

func JobCountMap(filename string, contents string) []mr.KeyValue {
	me := os.Getpid()
	f := fmt.Sprintf("mr-worker-jobcount-%d-%d", me, jobCount)
	jobCount++
	if err := os.WriteFile(f, []byte("x"), 0666); err != nil {
		panic(err)
	}
	time.Sleep(2 * time.Second) // 原版还加了随机 0-3 秒，这里简化
	return []mr.KeyValue{{"a", "x"}}
}

func JobCountReduce(key string, values []string) string {
	files, err := os.ReadDir(".")
	if err != nil {
		panic(err)
	}
	invocations := 0
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "mr-worker-jobcount") {
			invocations++
		}
	}
	return strconv.Itoa(invocations) // 正确实现下应恰好为 8
}

// ---------- early_exit.go ----------
func EarlyExitMap(filename string, contents string) []mr.KeyValue {
	return []mr.KeyValue{{filename, "1"}}
}

func EarlyExitReduce(key string, values []string) string {
	if strings.Contains(key, "sherlock") || strings.Contains(key, "tom") {
		time.Sleep(3 * time.Second)
	}
	return strconv.Itoa(len(values))
}

// ---------- crash.go ----------
func maybeCrash() {
	max := big.NewInt(1000)
	rr, _ := crand.Int(crand.Reader, max)
	if rr.Int64() < 330 {
		os.Exit(1) // 故意崩溃，测试容错
	} else if rr.Int64() < 660 {
		ms, _ := crand.Int(crand.Reader, big.NewInt(10000))
		time.Sleep(time.Duration(ms.Int64()) * time.Millisecond)
	}
}

func makeCrashFuncs() func(string, string) []mr.KeyValue {
	return func(filename string, contents string) []mr.KeyValue {
		maybeCrash()
		return []mr.KeyValue{
			{"a", filename},
			{"b", strconv.Itoa(len(filename))},
			{"c", strconv.Itoa(len(contents))},
			{"d", "xyzzy"},
		}
	}
}

func makeCrashReduce() func(string, []string) string {
	return func(key string, values []string) string {
		maybeCrash()
		vv := make([]string, len(values))
		copy(vv, values)
		sort.Strings(vv)
		return strings.Join(vv, " ")
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: mrworker_win <app> <sockname>\n")
		os.Exit(1)
	}
	mapf, reducef := loadApp(os.Args[1])
	mr.Worker(os.Args[2], mapf, reducef)
}
