// package kvraft

// import (
// 	"../labgob"
// 	"../labrpc"
// 	"log"
// 	"../raft"
// 	"sync"
// 	"sync/atomic"
// 	"time"
// )

// const Debug = 1

// const TIMEOUT = time.Millisecond * 500

// func DPrintf(format string, a ...interface{}) (n int, err error) {
// 	if Debug > 0 {
// 		log.Printf(format, a...)
// 	}
// 	return
// }

// type ReplyMsg struct {
// 	err   Err
// 	value string
// }


// type Op struct {
// 	// Your definitions here.
// 	// Field names must start with capital letters,
// 	// otherwise RPC will break.
// 	Key   string
// 	Value string
// 	Op    string

// 	OpId  int
// 	LeaderId int
// }

// type KVServer struct {
// 	mu      sync.Mutex
// 	me      int
// 	rf      *raft.Raft
// 	applyCh chan raft.ApplyMsg
// 	dead    int32 // set by Kill()

// 	maxraftstate int // snapshot if log grows this big

// 	// Your definitions here.
// 	killCh  chan struct{}

// 	latestOpId  int 

// 	database map[string] string
// 	op2chan map[int] chan ReplyMsg


	

// }


// func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
// 	// Your code here.

	
// 	_, isLeader := kv.rf.GetState()

// 	if !isLeader {
// 		reply.Err = ErrWrongLeader
// 		return
// 	}
// 	kv.mu.Lock()
// 	op := Op {
// 		Key: args.Key,
// 		Op: "Get",
// 		OpId: kv.latestOpId,
// 		LeaderId: kv.me,
// 	}

// 	kv.latestOpId++
// 	kv.mu.Unlock()

// 	reply.Err, reply.Value = kv.waitForCmd(op)
	
// }



// func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
// 	// Your code here.
	
// 	_, isLeader := kv.rf.GetState()

// 	if !isLeader {
// 		reply.Err = ErrWrongLeader
// 		return
// 	}
// 	kv.mu.Lock()
// 	op := Op {
// 		Key: args.Key,
// 		Value: args.Value,
// 		Op: args.Op,
// 		OpId: kv.latestOpId,
// 		LeaderId: kv.me,
// 	}

// 	kv.latestOpId++
// 	kv.mu.Unlock()

// 	reply.Err, _ = kv.waitForCmd(op)
// }

// func (kv *KVServer) waitForCmd(op Op) (Err, string) {

// 	_, _, isLeader := kv.rf.Start(op)
	
// 	if !isLeader {
// 		return ErrWrongLeader, ""
// 	}

// 	ch := kv.makeReplyCh(op);



// 	timeout := time.NewTimer(TIMEOUT)

// 	defer timeout.Stop()

// 	select {
// 	case res := <-ch:
// 		kv.clearCh(op.OpId)
// 		return res.err, res.value
// 	case <-timeout.C:
// 		kv.clearCh(op.OpId)
// 		return ErrTimeOut, ""
// 	}
// }

// func (kv *KVServer) makeReplyCh(op Op) chan ReplyMsg {
// 	kv.mu.Lock()
// 	defer kv.mu.Unlock()
// 	ch := make(chan ReplyMsg)
// 	kv.op2chan[op.OpId] = ch
// 	return ch
// }



// func (kv *KVServer) clearCh(opId int) {
// 	kv.mu.Lock()
// 	defer kv.mu.Unlock()
// 	delete(kv.op2chan, opId)
// }

// func (kv *KVServer) processApplyCh() {
// 	for !kv.killed() {
// 		select {
// 		case <- kv.killCh:
// 			DPrintf("server %v is killed\n", kv.me)
// 			return
// 		case applymsg := <- kv.applyCh:
// 			if !applymsg.CommandValid {
// 				continue
// 			}

// 			cmd := applymsg.Command.(Op)
// 			var err Err
// 			var value string
			
// 			kv.mu.Lock()

// 			switch cmd.Op {
// 			case "Get":
// 				value, err = kv.kvGet(cmd.Key)
// 				DPrintf("server %v get key %v\n", kv.me, cmd.Key)
// 			case "Put":
// 				value, err = kv.kvPut(cmd.Key, cmd.Value)
// 				DPrintf("server %v put key %v value %v\n", kv.me, cmd.Key, cmd.Value)
// 			case "Append":
// 				value, err = kv.kvAppend(cmd.Key, cmd.Value)
// 				DPrintf("server %v append key %v value %v\n", kv.me, cmd.Key, cmd.Value)
// 			}


// 			if replyClientCh, ok := kv.getReplyCh(cmd); ok { 
// 				kv.mu.Unlock()
// 				DPrintf("server %v reply to client\n", kv.me)
// 				replyClientCh <- ReplyMsg {
// 					err: err,
// 					value: value,
// 				}
// 			} else {
// 				kv.mu.Unlock()
// 			}
// 		}
// 	}
// }

// func (kv *KVServer) getReplyCh(op Op) (chan ReplyMsg, bool) {
// 	if op.LeaderId == kv.me {
// 		replyClientCh, ok := kv.op2chan[op.OpId] 
// 		return replyClientCh, ok
// 	} else {
// 		return nil, false
// 	}
// }

// func (kv *KVServer) kvGet(key string) (string, Err) {
// 	if value, ok := kv.database[key]; ok {
// 		return value, OK
// 	} else {
// 		return "", ErrNoKey
// 	}
// }

// func (kv *KVServer) kvPut(key string, value string) (string, Err) {
// 	kv.database[key] = value
// 	return "", OK
// }

// func (kv *KVServer) kvAppend(key string, value string) (string, Err) {
// 	v, _ := kv.kvGet(key)
// 	kv.database[key] = v + key
// 	return "", OK
// }

// //
// // the tester calls Kill() when a KVServer instance won't
// // be needed again. for your convenience, we supply
// // code to set rf.dead (without needing a lock),
// // and a killed() method to test rf.dead in
// // long-running loops. you can also add your own
// // code to Kill(). you're not required to do anything
// // about this, but it may be convenient (for example)
// // to suppress debug output from a Kill()ed instance.
// //
// func (kv *KVServer) Kill() {
// 	atomic.StoreInt32(&kv.dead, 1)
// 	kv.rf.Kill()
// 	// Your code here, if desired.
// 	close(kv.killCh)
// }

// func (kv *KVServer) killed() bool {
// 	z := atomic.LoadInt32(&kv.dead)
// 	return z == 1
// }

// //
// // servers[] contains the ports of the set of
// // servers that will cooperate via Raft to
// // form the fault-tolerant key/value service.
// // me is the index of the current server in servers[].
// // the k/v server should store snapshots through the underlying Raft
// // implementation, which should call persister.SaveStateAndSnapshot() to
// // atomically save the Raft state along with the snapshot.
// // the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// // in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// // you don't need to snapshot.
// // StartKVServer() must return quickly, so it should start goroutines
// // for any long-running work.
// //
// func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
// 	// call labgob.Register on structures you want
// 	// Go's RPC library to marshall/unmarshall.
// 	labgob.Register(Op{})

// 	kv := new(KVServer)
// 	kv.me = me
// 	kv.maxraftstate = maxraftstate

// 	// You may need initialization code here.

// 	kv.applyCh = make(chan raft.ApplyMsg)
// 	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

// 	// You may need initialization code here.

// 	kv.latestOpId = 0 
	
// 	// database
// 	kv.database = make(map[string] string)

// 	// OpId to channel
// 	kv.op2chan = make(map[int] chan ReplyMsg)
	
// 	go kv.processApplyCh()

// 	return kv
// }



package kvraft

import (
	"fmt"
	"../labgob"
	"../labrpc"
	"log"
	"../raft"
	"sync"
	"time"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

const WaitCmdTimeOut = time.Millisecond * 500 

const Debug = 0

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	MsgId    msgId
	ReqId    int64
	ClientId int64
	Key      string
	Value    string
	Method   string
}

type NotifyMsg struct {
	Err   Err
	Value string
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	stopCh  chan struct{}

	maxraftstate int // snapshot if log grows this big
	// Your definitions here.
	msgNotify   map[int64]chan NotifyMsg
	lastApplies map[int64]msgId // last apply put/append msg
	data        map[string]string

	persister      *raft.Persister
	lastApplyIndex int
	lastApplyTerm  int
}


func (kv *KVServer) dataGet(key string) (err Err, val string) {
	if v, ok := kv.data[key]; ok {
		err = OK
		val = v
		return
	} else {
		err = ErrNoKey
		return
	}
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {

	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	op := Op{
		MsgId:    args.MsgId,
		ReqId:    nrand(),
		Key:      args.Key,
		Method:   "Get",
		ClientId: args.ClientId,
	}
	res := kv.waitCmd(op)
	reply.Err = res.Err
	reply.Value = res.Value
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	op := Op{
		MsgId:    args.MsgId,
		ReqId:    nrand(),
		Key:      args.Key,
		Value:    args.Value,
		Method:   args.Op,
		ClientId: args.ClientId,
	}
	reply.Err = kv.waitCmd(op).Err
}

func (kv *KVServer) removeCh(id int64) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	delete(kv.msgNotify, id)
}

func (kv *KVServer) waitCmd(op Op) (res NotifyMsg) {

	_, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		res.Err = ErrWrongLeader
		return
	}

	kv.mu.Lock()
	ch := make(chan NotifyMsg, 1)
	kv.msgNotify[op.ReqId] = ch
	kv.mu.Unlock()
	t := time.NewTimer(WaitCmdTimeOut)
	defer t.Stop()
	select {
	case res = <-ch:
		kv.removeCh(op.ReqId)
		return
	case <-t.C:
		kv.removeCh(op.ReqId)
		res.Err = ErrTimeOut
		return
	}
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) Kill() {
	kv.rf.Kill()
	close(kv.stopCh)
	// Your code here, if desired.
}

func (kv *KVServer) isRepeated(clientId int64, id msgId) bool {
	if val, ok := kv.lastApplies[clientId]; ok {
		return val == id
	}
	return false
}

func (kv *KVServer) waitApplyCh() {
	for {
		select {
		case <-kv.stopCh:
			return
		case msg := <-kv.applyCh:
			if !msg.CommandValid {
				continue
			}
			op := msg.Command.(Op)
			kv.mu.Lock()

			isRepeated := kv.isRepeated(op.ClientId, op.MsgId)
			switch op.Method {
			case "Put":
				if !isRepeated {
					kv.data[op.Key] = op.Value
					kv.lastApplies[op.ClientId] = op.MsgId
				}
			case "Append":
				if !isRepeated {
					_, v := kv.dataGet(op.Key)
					kv.data[op.Key] = v + op.Value
					kv.lastApplies[op.ClientId] = op.MsgId
				}

			case "Get":
			default:
				panic(fmt.Sprintf("unknown method: %s", op.Method))
			}
			
			if ch, ok := kv.msgNotify[op.ReqId]; ok {
				_, v := kv.dataGet(op.Key)
				ch <- NotifyMsg{
					Err:   OK,
					Value: v,
				}
			}
			kv.mu.Unlock()
		}
	}
}



func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.persister = persister

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.

	kv.msgNotify = make(map[int64]chan NotifyMsg)
	kv.data = make(map[string]string)
	kv.lastApplies = make(map[int64]msgId)

	kv.stopCh = make(chan struct{})

	go kv.waitApplyCh()


	return kv
}