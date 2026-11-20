package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type TaskState int

const (
	Unassigned TaskState = iota
	InProgress
	Completed
)

type TaskInfo struct {
	State     TaskState
	StartTime time.Time
}

type Coordinator struct {
	// Your definitions here.
	mu          sync.Mutex
	mapTasks    []TaskInfo
	reduceTasks []TaskInfo
	nReduce     int
	files       []string
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) RequestTask(args *RequestTaskArgs, reply *RequestTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// Check for unassigned map tasks
	allMapDone := true
	for i, task := range c.mapTasks {
		if task.State != Completed {
			allMapDone = false
		}

		if task.State == Unassigned || (task.State == InProgress && time.Since(task.StartTime) > 10*time.Second) {
			c.mapTasks[i].State = InProgress
			c.mapTasks[i].StartTime = time.Now()
			reply.TaskType = MapTask
			reply.TaskId = i
			reply.FileName = c.files[i]
			reply.NMap = len(c.files)
			reply.NReduce = c.nReduce
			return nil
		}

	}

	if !allMapDone {
		reply.TaskType = WaitTask
		return nil
	}
	allReduceDone := true
	for i, task := range c.reduceTasks {
		if task.State != Completed {
			allReduceDone = false
		}
		if task.State == Unassigned || (task.State == InProgress && time.Since(task.StartTime) > 10*time.Second) {
			c.reduceTasks[i].State = InProgress
			c.reduceTasks[i].StartTime = time.Now()
			reply.TaskType = ReduceTask
			reply.TaskId = i
			reply.NMap = len(c.files)
			reply.NReduce = c.nReduce
			return nil
		}
	}
	if !allReduceDone {
		reply.TaskType = WaitTask
		return nil
	}
	reply.TaskType = ExitTask
	return nil
}

func (c *Coordinator) ReportTask(args *ReportTaskArgs, reply *ReportTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if args.TaskType == MapTask {
		if c.mapTasks[args.TaskId].State != Completed { // 防抖：如果已经标记了完成，就不要重复标记了
			c.mapTasks[args.TaskId].State = Completed
		}
	} else if args.TaskType == ReduceTask {
		if c.reduceTasks[args.TaskId].State != Completed {
			c.reduceTasks[args.TaskId].State = Completed
		}
	}
	return nil // 汇报完毕，不用返回数据
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
// func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
// 	reply.Y = args.X + 1
// 	return nil
// }

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server(sockname string) {
	rpc.Register(c)
	rpc.HandleHTTP()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatalf("listen error %s: %v", sockname, e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, task := range c.reduceTasks {
		if task.State != Completed {
			return false
		}
	}
	return true
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(sockname string, files []string, nReduce int) *Coordinator {
	c := Coordinator{
		files:       files,
		nReduce:     nReduce,
		mapTasks:    make([]TaskInfo, len(files)),
		reduceTasks: make([]TaskInfo, nReduce),
	}

	// Your code here.
	for i := range c.mapTasks {
		c.mapTasks[i].State = Unassigned
	}
	for i := range c.reduceTasks {
		c.reduceTasks[i].State = Unassigned
	}

	c.server(sockname)
	return &c
}
