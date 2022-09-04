package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

type Args struct {
	X string
}

type Job struct {
	// 工作类型，0为wait等待，1为map任务，2为reduce任务
	jobtype int
	// 任务文件
	files []string
	// 任务号，对于map任务用于生成存储中间kv的文件名，而对于reduce任务用于获取文件名和生成最终输出文件名
	Xth int

	nreduce int
}


// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the master.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func masterSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
