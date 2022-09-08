package mr

import "log"
import "net"
import "os"
import "net/rpc"
import "net/http"
import "fmt"
import "time"
import "sync"



type Master struct {
	// Your definitions here.
	
	// these two variables are only readable
	nm int
	nr int

	// protect 
	mutex sync.Mutex

	status PhaseType

	MapChannel chan *Job
	ReduceChannel chan *Job

	mapJobInfo [] JobInfo
	reduceJobInfo [] JobInfo

}


// Your code here -- RPC handlers for the worker to call.

func (m *Master) Handler(request *Request, reply *Job) error {
	m.mutex.Lock()
	if request.RQ == ACQUIRE {

		switch m.status {

		case MAP_PHASE:
			if len(m.MapChannel) > 0 {
				*reply = *<-m.MapChannel
				m.mapJobInfo[reply.Xth].jobstatus = PROCESSING
				m.mapJobInfo[reply.Xth].start = time.Now()
				fmt.Println("fire the map job:", *reply)
			} else {
				reply.JobType = WAIT_TASK
			}
			

		case REDUCE_PHASE:
			if len(m.ReduceChannel) > 0 {
				*reply = *<-m.ReduceChannel
				m.reduceJobInfo[reply.Xth].jobstatus = PROCESSING
				m.reduceJobInfo[reply.Xth].start = time.Now()
				fmt.Println("shoot the reduce job:", *reply)
			} else {
				reply.JobType = WAIT_TASK
			}
			

		case FINISH_PHASE:
			reply.JobType = KILL_TASK

		default:
			fmt.Println("undefined phase in ACQUIRE ...")

		}

	} else if request.RQ == SUBMIT {

		switch m.status {

		case MAP_PHASE:
			// 多此一举
			if request.JobType == MAP_PHASE {
				fmt.Println("receive the map job:", request.Xth)
				m.mapJobInfo[request.Xth].jobstatus = FINISHED
				m.CheckAndGrow()
			}

		case REDUCE_PHASE:
			// only receive REDUCE_PHASE, to avoid crash worker send its task
			if request.JobType == REDUCE_PHASE {
				fmt.Println("get the reduce job:", request.Xth)
				m.reduceJobInfo[request.Xth].jobstatus = FINISHED
				m.CheckAndGrow()
			}

		case FINISH_PHASE:
			//
			reply.JobType = KILL_TASK

		default:
			fmt.Println("undefined phase in SUBMIT ...")
		}

	} else {
		fmt.Println("undefined request")
	}
		
	m.mutex.Unlock()
	return nil
}

// this function work as a coroutine, responsible for reshoot the task which is under processing over 10 seconds
func (m *Master) CrashHandler() {
	for {
		// check each second
		time.Sleep(1 * time.Second)
		m.mutex.Lock()
		switch m.status {
		
		case MAP_PHASE:
			for i := 0; i < m.nm; i++ {
				// over time, treat as crash
				if m.mapJobInfo[i].jobstatus == PROCESSING && time.Now().Sub(m.mapJobInfo[i].start) >= 10 * time.Second {
					// change the job status and reinput to channel
					fmt.Println("recovery map crash:", i)
					m.mapJobInfo[i].jobstatus = WAITING
					m.MapChannel <- m.mapJobInfo[i].jobptr
				}
			}

		case REDUCE_PHASE:
			for i := 0; i < m.nr; i++ {
				// over time, treat as crash
				if m.reduceJobInfo[i].jobstatus == PROCESSING && time.Now().Sub(m.reduceJobInfo[i].start) >= 10 * time.Second {
					// change the job status and reinput to channel
					fmt.Println("recovery reduce crash:", i)
					m.reduceJobInfo[i].jobstatus = WAITING
					m.ReduceChannel <- m.reduceJobInfo[i].jobptr
				}
			}
			
		case FINISH_PHASE:
			fmt.Println("is over")
			m.mutex.Unlock()
			break

		default:
			fmt.Println("undefined phase")
		} 

		m.mutex.Unlock()
	}

}

// put all maptask into channel
func (m *Master) GenerateMapTask(files []string) {
	
	m.MapChannel = make(chan *Job, m.nm)
	m.mapJobInfo = make([]JobInfo, m.nm)
	
	for i := 0; i < m.nm; i++ {
		mj := Job {
			JobType: MAP_TASK,
			Files: []string {files[i]},
			Xth: i,
			Nreduce: m.nr, 
		}
		fmt.Println("make map job:", mj)
		m.mapJobInfo[i] = JobInfo {
			jobstatus: WAITING,
			jobptr: &mj,
		}
		fmt.Println("job infomation:", m.mapJobInfo[i])
		m.MapChannel <- &mj
	}
}

// put all reduce into channel
func (m *Master) GenerateReduceTask() {

	m.ReduceChannel = make(chan *Job, m.nr)
	m.reduceJobInfo = make([]JobInfo, m.nr)

	for i := 0; i < m.nr; i++ {
		tasks := make([]string, m.nm)
		for j := 0; j < m.nm; j++ {
			tasks[j] = fmt.Sprintf("%s%d%s%d", "mr-", j, "-", i)
		}
		rj := Job {
			JobType: REDUCE_TASK,
			Files: tasks[:m.nm],
			Xth: i,
			Nreduce: m.nr, 
		}
		fmt.Println("make reduce job:", rj)
		m.reduceJobInfo[i] = JobInfo {
			jobstatus: WAITING,
			jobptr: &rj,
		}
		fmt.Println("job infomation:", m.reduceJobInfo[i])
		m.ReduceChannel <- &rj
	}
}

func (m *Master) CheckAndGrow() {
	if m.status == MAP_PHASE {
		for i := 0; i < m.nm; i++ {
			if m.mapJobInfo[i].jobstatus != FINISHED {
				return
			}
		}
		// all of the map task status is  FINISHED, we prepare the reduce job and grow the status to REDUCE_PHASE
		fmt.Println("master grows status to REDUCE_PHASE")
		m.GenerateReduceTask();
		m.status = REDUCE_PHASE
	} else if m.status == REDUCE_PHASE {
		for i := 0; i < m.nr; i++ {
			if m.reduceJobInfo[i].jobstatus != FINISHED {
				return
			}
		}
		// all of the reduce task status is FINISHED, we can grow the status to FINISH
		m.status = FINISH_PHASE
		fmt.Println("master grows status to FINISH")
	} else {
		fmt.Println("master status is already FINISH")
	}
}

// //
// // an example RPC handler.
// //
// // the RPC argument and reply types are defined in rpc.go.
// //
// func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
// 	reply.Y = args.X + 1
// 	return nil
// }


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
	if m.status == FINISH_PHASE {
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
	m.nm = len(files)
	m.nr = nReduce
	m.status = MAP_PHASE
	m.GenerateMapTask(files)

	go m.CrashHandler()


	m.server()
	return &m
}


