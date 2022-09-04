package mr

import "log"
import "net"
import "os"
import "net/rpc"
import "net/http"


type Master struct {
	// Your definitions here.

	// record all of the task file
	files []string
	// record the task we have finished
	have_finish []string
	// record the task and worker map relationship
	task_toworker map[string]int

}

// Your code here -- RPC handlers for the worker to call.

func (m *Master) Handler(args *Args, reply *Job) error {
	return nil
}


//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}


//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	ret := false

	// Your code here.
	if len(m.have_finish) == len(m.files) {
		ret = true
	}

	return ret
}

//
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}

	// Your code here.
	m.files = files


	m.server()
	return &m
}
