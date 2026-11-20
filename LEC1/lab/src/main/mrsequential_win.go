package main

// Windows 适配版 mrsequential：把各应用的 Map/Reduce 直接编译进来，
// 不依赖 -buildmode=plugin（Windows 不支持）。
// 用法：mrsequential_win <app> pg*.txt → 生成 mr-out-0（顺序版标准答案）
//   app 可以是 wc / indexer / early_exit / rtiming 等

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"6.5840/mr"
)

func wcMap(filename string, contents string) []mr.KeyValue {
	ff := func(r rune) bool { return !unicode.IsLetter(r) }
	words := strings.FieldsFunc(contents, ff)
	kva := []mr.KeyValue{}
	for _, w := range words {
		kva = append(kva, mr.KeyValue{w, "1"})
	}
	return kva
}

func wcReduce(key string, values []string) string {
	return strconv.Itoa(len(values))
}

// 按应用名返回对应的 Map/Reduce 函数（与 worker 的 loadApp 保持一致）。
func loadAppSeq(name string) (func(string, string) []mr.KeyValue, func(string, []string) string) {
	switch name {
	case "nocrash":
		return noCrashMap, noCrashReduce
	case "indexer":
		return indexerMap, indexerReduce
	case "early_exit":
		return earlyExitMap, earlyExitReduce
	case "rtiming":
		return rtimingMap, rtimingReduce
	default: // wc / nocrash / mtiming
		return wcMap, wcReduce
	}
}

func noCrashMap(filename string, contents string) []mr.KeyValue {
	return []mr.KeyValue{
		{"a", filename},
		{"b", strconv.Itoa(len(filename))},
		{"c", strconv.Itoa(len(contents))},
		{"d", "xyzzy"},
	}
}

func noCrashReduce(key string, values []string) string {
	vv := make([]string, len(values))
	copy(vv, values)
	sort.Strings(vv)
	return strings.Join(vv, " ")
}

func indexerMap(document string, value string) (res []mr.KeyValue) {
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

func indexerReduce(key string, values []string) string {
	sort.Strings(values)
	return fmt.Sprintf("%d %s", len(values), strings.Join(values, ","))
}

func earlyExitMap(filename string, contents string) []mr.KeyValue {
	return []mr.KeyValue{{filename, "1"}}
}

func earlyExitReduce(key string, values []string) string {
	return strconv.Itoa(len(values))
}

func rtimingMap(filename string, contents string) []mr.KeyValue {
	return []mr.KeyValue{
		{"a", "1"}, {"b", "1"}, {"c", "1"}, {"d", "1"}, {"e", "1"},
		{"f", "1"}, {"g", "1"}, {"h", "1"}, {"i", "1"}, {"j", "1"},
	}
}

func rtimingReduce(key string, values []string) string {
	return fmt.Sprintf("%d 2", len(values))
}

type ByKey []mr.KeyValue

func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: mrsequential_win <app> pg*.txt\n")
		os.Exit(1)
	}
	mapf, reducef := loadAppSeq(os.Args[1])
	intermediate := []mr.KeyValue{}
	for _, filename := range os.Args[2:] {
		content, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		intermediate = append(intermediate, mapf(filename, string(content))...)
	}
	sort.Sort(ByKey(intermediate))

	ofile, err := os.Create("mr-out-0")
	if err != nil {
		panic(err)
	}
	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, reducef(intermediate[i].Key, values))
		i = j
	}
	ofile.Close()
}
