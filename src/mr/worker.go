package mr

import "fmt"
import "log"
import "net/rpc"
import "hash/fnv"
import "os"
import "io/ioutil"
import "sort"
import "encoding/json"
import "time"


//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }



//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}


//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// uncomment to send the Example RPC to the master.
	// CallExample()

	// we beg master to allocate task 
	for {
		request := Request{}
		request.RQ = ACQUIRE

		job:= Job{}
		call("Master.Handler", &request, &job)
		
		switch job.JobType {
		case MAP_TASK:
			fmt.Println("get the map job:", job)
			DoMap(mapf, job)
		case REDUCE_TASK:
			DoReduce(reducef, job)
		case WAIT_TASK:
			fmt.Println("wait for 2 seconds ...")
			time.Sleep(2 * time.Second)
		case KILL_TASK:
			fmt.Println("no tasks, kill self ...")
			return
		default:
			fmt.Println("undefined job type ...")
		}
	}

}

func DoMap(mapf func(string, string) []KeyValue, job Job) {

	// check if it is the right task
	if job.JobType != MAP_TASK {
		fmt.Println("the job type should be map!!!")
		return
	}
	
	intermediate := []KeyValue{}
	for _, filename := range job.Files {
		// open and read the file correctly
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalf("cannot read %v", filename)
		}
		// close file
		file.Close()

		// now we get the content, call map function and sort the intermediate kv
		kva := mapf(filename, string(content))
		intermediate = append(intermediate, kva...)

	}
	sort.Sort(ByKey(intermediate))
	
	// prepare hashtable
	hashkey2kv := make([][]KeyValue, job.Nreduce)
  	for _, kv := range intermediate {
		hashkey2kv[ihash(kv.Key) % job.Nreduce] = append(hashkey2kv[ihash(kv.Key) % job.Nreduce], kv)
	}

	// write the intermediate kv into  each mr-X-Y file
	for i := 0; i < job.Nreduce; i++ {
		// create or open specified file
		oname := fmt.Sprintf("%s%d%s%d", "mr-", job.Xth, "-", i)
		temp := fmt.Sprintf("%d%s%d", job.Xth, "-", i)
		ofile, _ := ioutil.TempFile(".", temp)
		// write kv
		enc := json.NewEncoder(ofile)
		for _, kv := range hashkey2kv[i] {
			en_err := enc.Encode(&kv)
			if en_err != nil {
				log.Fatalf("json encode error")
			} 
		}
		// atomically rename
		os.Rename("./" + ofile.Name(), "./" + oname)
		// close file
		ofile.Close()
	}
	
	// tell the master that we finished the task
	request := Request{}
	request.RQ = SUBMIT
	request.JobType = MAP_TASK
	request.Xth = job.Xth

	call("Master.Handler", &request, &job)
}

func DoReduce(reducef func(string, []string) string, job Job) {

	// check if it is the right task
	if job.JobType != REDUCE_TASK {
		fmt.Println("the job type should be reduce!!!")
		return
	}

	kva := []KeyValue{}
	for _, filename := range job.Files {
		file, _ := os.Open(filename)
		dec := json.NewDecoder(file)
  		for {
  		  	var kv KeyValue
  		  	if err := dec.Decode(&kv); err != nil {
  		  		break
  		  	}
  		  	kva = append(kva, kv)
  		}
	}

	sort.Sort(ByKey(kva))

	oname := fmt.Sprintf("%s%d", "mr-out-", job.Xth)
	
	temp := fmt.Sprintf("%d", job.Xth)
	ofile, _ := ioutil.TempFile(".", temp)

	i := 0
	for i < len(kva) {
		j := i + 1
		for j < len(kva) && kva[j].Key == kva[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, kva[k].Value)
		}
		output := reducef(kva[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(ofile, "%v %v\n", kva[i].Key, output)

		i = j
	}

	// atomically rename
	os.Rename("./" + ofile.Name(), "./" + oname)
	// close file
	ofile.Close()

	// tell the master that we finished the task
	request := Request{}
	request.RQ = SUBMIT
	request.JobType = REDUCE_TASK
	request.Xth = job.Xth
	call("Master.Handler", &request, &job)

}

// //
// // example function to show how to make an RPC call to the master.
// //
// // the RPC argument and reply types are defined in rpc.go.
// //
// func CallExample() {

// 	// declare an argument structure.
// 	args := ExampleArgs{}

// 	// fill in the argument(s).
// 	args.X = 99

// 	// declare a reply structure.
// 	reply := ExampleReply{}

// 	// send the RPC request, wait for the reply.
// 	call("Master.Example", &args, &reply)

// 	// reply.Y should be 100.
// 	fmt.Printf("reply.Y %v\n", reply.Y)
// }

//
// send an RPC request to the master, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := masterSock()
	// c is *Client
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	// rpcname is server method to call, in example above is Master.Example
	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
