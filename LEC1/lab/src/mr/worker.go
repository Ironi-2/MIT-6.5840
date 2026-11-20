package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

var coordSockName string // socket for coordinator

// main/mrworker.go calls this function.
func Worker(sockname string, mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	coordSockName = sockname

	// 无限循环，不断向 Coordinator 请求任务
	for {
		args := RequestTaskArgs{}
		reply := RequestTaskReply{}

		ok := call("Coordinator.RequestTask", &args, &reply)
		if !ok {
			// 如果连不上 Coordinator，说明 Coordinator 已经退出，Worker 也直接退出
			return
		}

		switch reply.TaskType {
		case MapTask:
			doMap(reply, mapf)
		case ReduceTask:
			doReduce(reply, reducef)
		case WaitTask:
			// 没有任务可做，稍微休眠一下避免疯狂占用 CPU
			time.Sleep(50 * time.Millisecond)
		case ExitTask:
			// 所有任务完成，下班！
			return
		}
	}
}

// 执行 Map 任务的逻辑
func doMap(reply RequestTaskReply, mapf func(string, string) []KeyValue) {
	// 1. 读取文件内容
	content, err := os.ReadFile(reply.FileName)
	if err != nil {
		log.Fatalf("cannot read %v: %v", reply.FileName, err)
	}

	// 2. 调用用户自定义的 mapf 函数
	kva := mapf(reply.FileName, string(content))

	// 3. 将 KeyValue 按 hash 分桶（分成 NReduce 个桶）
	buckets := make([][]KeyValue, reply.NReduce)
	for _, kv := range kva {
		idx := ihash(kv.Key) % reply.NReduce
		buckets[idx] = append(buckets[idx], kv)
	}

	// 4. 将每个桶的数据写入临时文件，并原子重命名
	for i := 0; i < reply.NReduce; i++ {
		// 创建临时文件
		tempFile, err := os.CreateTemp(".", "mr-tmp-*")
		if err != nil {
			log.Fatalf("cannot create temp file: %v", err)
		}
		enc := json.NewEncoder(tempFile)

		for _, kv := range buckets[i] {
			if err := enc.Encode(&kv); err != nil {
				log.Fatalf("encode error: %v", err)
			}
		}
		tempFile.Close()

		// 格式为 mr-X-Y (X 是 Map 任务号，Y 是 Reduce 任务号)
		outName := fmt.Sprintf("mr-%d-%d", reply.TaskId, i)
		err = os.Rename(tempFile.Name(), outName)
		if err != nil {
			log.Fatalf("rename error: %v", err)
		}
	}

	// 5. 向 Coordinator 汇报任务完成
	reportArgs := ReportTaskArgs{TaskType: MapTask, TaskId: reply.TaskId}
	reportReply := ReportTaskReply{}
	call("Coordinator.ReportTask", &reportArgs, &reportReply)
}

// 执行 Reduce 任务的逻辑
func doReduce(reply RequestTaskReply, reducef func(string, []string) string) {
	// 1. 读取所有 Map 任务生成的属于该 Reduce 任务的中间文件
	kvs := make(map[string][]string)
	for i := 0; i < reply.NMap; i++ {
		filename := fmt.Sprintf("mr-%d-%d", i, reply.TaskId)
		file, err := os.Open(filename)
		if err != nil {
			// 如果文件不存在，可能对应的 map 任务还没做完或超时，忽略
			continue
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err == io.EOF {
				break
			} else if err != nil {
				log.Fatalf("decode error: %v", err)
			}
			kvs[kv.Key] = append(kvs[kv.Key], kv.Value)
		}
		file.Close()
	}

	// 2. 对所有的 key 进行排序，保证输出有序
	var keys []string
	for k := range kvs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. 创建临时文件，写入最终结果
	tempFile, err := os.CreateTemp(".", "mr-out-tmp-*")
	if err != nil {
		log.Fatalf("cannot create temp file: %v", err)
	}
	for _, k := range keys {
		output := reducef(k, kvs[k])
		fmt.Fprintf(tempFile, "%v %v\n", k, output)
	}
	tempFile.Close()

	// 4. 原子重命名为最终输出文件
	outName := fmt.Sprintf("mr-out-%d", reply.TaskId)
	err = os.Rename(tempFile.Name(), outName)
	if err != nil {
		log.Fatalf("rename error: %v", err)
	}

	// 5. 向 Coordinator 汇报任务完成
	reportArgs := ReportTaskArgs{TaskType: ReduceTask, TaskId: reply.TaskId}
	reportReply := ReportTaskReply{}
	call("Coordinator.ReportTask", &reportArgs, &reportReply)
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	c, err := rpc.DialHTTP("unix", coordSockName)
	if err != nil {
		// log.Fatal("dialing:", err)
		return false // 删掉 log.Fatal，因为如果 Coordinator 挂了，Worker 应该安静地退出
	}
	defer c.Close()

	if err := c.Call(rpcname, args, reply); err == nil {
		return true
	}
	return false
}
