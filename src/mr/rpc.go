package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"
import "time"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

// type ExampleArgs struct {
// 	X int
// }

// type ExampleReply struct {
// 	Y int
// }

// Add your RPC definitions here.

// define task type
type TaskType int 
const (
	MAP_TASK = iota
	REDUCE_TASK
	WAIT_TASK
	KILL_TASK
)

// define task type
type RequestType int 
const (
	ACQUIRE = iota
	SUBMIT
)

// define master phase type
type PhaseType int 
const (
	MAP_PHASE = iota
	REDUCE_PHASE
	FINISH_PHASE
)

type JobStatus int
const (
	WAITING = iota
	PROCESSING
	FINISHED
)

type Job struct {
	JobType TaskType
	Files   []string
	Xth     int
	Nreduce      int
}

type JobInfo struct {
	jobstatus JobStatus
	start time.Time
	jobptr *Job
}

type Request struct {
	RQ RequestType
	JobType TaskType
	Xth     int
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
