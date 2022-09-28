package kvraft

import "../labrpc"
import "crypto/rand"
import "math/big"


type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	clientId int64
	leaderId int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	// You'll have to add code here.
	ck.clientId = nrand()
	ck.leaderId = 0
	return ck
}

func (ck *Clerk) Get(key string) string {
	i:= ck.leaderId
	args := ck.makeGetArgs(key)
		reply := ck.makeGetReply()
	for {
		
		ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
		if !ok {
			DPrintf("client %v call KVServer.Get not ok, key %v from leader %v\n", ck.clientId, key, i)
			i = (i + 1) % len(ck.servers)
			continue
		}

		if ok {
			switch reply.Err {
			case OK :
				DPrintf("client %v get key %v ok from leader %v\n", ck.clientId, key, i)
				ck.leaderId = i
				return reply.Value
			case ErrNoKey :
				DPrintf("client %v get key %v do not have this key from leader %v\n", ck.clientId, key, i)
				ck.leaderId = i
				return ""
			case ErrWrongLeader :
				i = (i + 1) % len(ck.servers)
				continue
			case ErrTimeOut :
				DPrintf("client %v get key %v timeout from leader %v\n", ck.clientId, key, i)
				// i = (i + 1) % len(ck.servers)
				continue
			default :
				i = (i + 1) % len(ck.servers)
				continue
			}
		}
	}
}

func (ck *Clerk) makeGetArgs(key string) GetArgs {
	return GetArgs {
		Key: key,
		
		MsgId: ck.genMsgId(), 
		ClientId: ck.clientId,
	}
}

func (ck *Clerk) makeGetReply() GetReply {
	return GetReply {
		Err: "",
		Value: "",
	}
}



func (ck *Clerk) PutAppend(key string, value string, op string) {
	i:= ck.leaderId
	args := ck.makePutAppendArgs(key, value, op)
	reply := ck.makePutAppendReply()
	for {
		ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)

		if !ok {
			DPrintf("client %v call KVServer.PutAppend not ok, key %v value %v from leader %v\n", ck.clientId, key, value, i)
			i = (i + 1) % len(ck.servers)
			continue
		}

		if ok {
			switch reply.Err {
			case OK :
				DPrintf("client %v %v key %v value %v ok from leader %v\n", ck.clientId, op, key, value, i)
				return
			case ErrNoKey :
				DPrintf("client %v %v key %v value %v not have this key, panic!!!!!!!!!!!!! from leader %v\n", ck.clientId, op, key, value, i)
				for {}
			case ErrWrongLeader :
				i = (i + 1) % len(ck.servers)
				continue
			case ErrTimeOut :
				DPrintf("client %v %v key %v value %v timeout from leader %v\n", ck.clientId, op, key, value, i)
				continue
			default :
				i = (i + 1) % len(ck.servers)
				continue
			}
		}
	}
}

func (ck *Clerk) makePutAppendArgs(key string, value string, op string) PutAppendArgs {
	return PutAppendArgs {
		Key: key,
		Value: value,
		Op: op,

		MsgId: ck.genMsgId(), 
		ClientId: ck.clientId,
	}
}

func (ck *Clerk) makePutAppendReply() PutAppendReply {
	return PutAppendReply {
		Err: "",
	}
}

func (ck *Clerk) genMsgId() msgId {
	return msgId(nrand())
}


func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}



// package kvraft

// import (
// 	"../labrpc"
// 	"log"
// 	"time"
// )
// import "crypto/rand"
// import "math/big"

// const (
// 	ChangeLeaderInterval = time.Millisecond * 20
// )

// type Clerk struct {
// 	servers []*labrpc.ClientEnd
// 	// You will have to modify this struct.
// 	clientId int64
// 	leaderId int
// }

// func nrand() int64 {
// 	max := big.NewInt(int64(1) << 62)
// 	bigx, _ := rand.Int(rand.Reader, max)
// 	x := bigx.Int64()
// 	return x
// }

// func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
// 	ck := new(Clerk)
// 	ck.servers = servers
// 	ck.clientId = nrand()
// 	// You'll have to add code here.
// 	return ck
// }

// func (ck *Clerk) genMsgId() msgId {
// 	return msgId(nrand())
// }



// func (ck *Clerk) Get(key string) string {
// 	i:= ck.leaderId
// 	args := ck.makeGetArgs(key)
// 		reply := ck.makeGetReply()
// 	for {
		
// 		ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
// 		if !ok {
// 			DPrintf("client %v call KVServer.Get not ok, key %v from leader %v\n", ck.clientId, key, i)
// 			i = (i + 1) % len(ck.servers)
// 			continue
// 		}

// 		if ok {
// 			switch reply.Err {
// 			case OK :
// 				DPrintf("client %v get key %v ok from leader %v\n", ck.clientId, key, i)
// 				ck.leaderId = i
// 				return reply.Value
// 			case ErrNoKey :
// 				DPrintf("client %v get key %v do not have this key from leader %v\n", ck.clientId, key, i)
// 				ck.leaderId = i
// 				return ""
// 			case ErrWrongLeader :
// 				i = (i + 1) % len(ck.servers)
// 				continue
// 			case ErrTimeOut :
// 				DPrintf("client %v get key %v timeout from leader %v\n", ck.clientId, key, i)
// 				// i = (i + 1) % len(ck.servers)
// 				continue
// 			default :
// 				i = (i + 1) % len(ck.servers)
// 				continue
// 			}
// 		}
// 	}
// }

// func (ck *Clerk) makeGetArgs(key string) GetArgs {
// 	return GetArgs {
// 		Key: key,
		
// 		MsgId: ck.genMsgId(), 
// 		ClientId: ck.clientId,
// 	}
// }

// func (ck *Clerk) makeGetReply() GetReply {
// 	return GetReply {
// 		Err: "",
// 		Value: "",
// 	}
// }



// func (ck *Clerk) PutAppend(key string, value string, op string) {
// 	i:= ck.leaderId
// 	args := ck.makePutAppendArgs(key, value, op)
// 	reply := ck.makePutAppendReply()
// 	for {
// 		ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)

// 		if !ok {
// 			DPrintf("client %v call KVServer.PutAppend not ok, key %v value %v from leader %v\n", ck.clientId, key, value, i)
// 			i = (i + 1) % len(ck.servers)
// 			continue
// 		}

// 		if ok {
// 			switch reply.Err {
// 			case OK :
// 				DPrintf("client %v %v key %v value %v ok from leader %v\n", ck.clientId, op, key, value, i)
// 				return
// 			case ErrNoKey :
// 				DPrintf("client %v %v key %v value %v not have this key, panic!!!!!!!!!!!!! from leader %v\n", ck.clientId, op, key, value, i)
// 				for {}
// 			case ErrWrongLeader :
// 				i = (i + 1) % len(ck.servers)
// 				continue
// 			case ErrTimeOut :
// 				DPrintf("client %v %v key %v value %v timeout from leader %v\n", ck.clientId, op, key, value, i)
// 				continue
// 			default :
// 				i = (i + 1) % len(ck.servers)
// 				continue
// 			}
// 		}
// 	}
// }

// func (ck *Clerk) makePutAppendArgs(key string, value string, op string) PutAppendArgs {
// 	return PutAppendArgs {
// 		Key: key,
// 		Value: value,
// 		Op: op,

// 		MsgId: ck.genMsgId(), 
// 		ClientId: ck.clientId,
// 	}
// }

// func (ck *Clerk) makePutAppendReply() PutAppendReply {
// 	return PutAppendReply {
// 		Err: "",
// 	}
// }

// func (ck *Clerk) Put(key string, value string) {
// 	ck.PutAppend(key, value, "Put")
// }
// func (ck *Clerk) Append(key string, value string) {
// 	ck.PutAppend(key, value, "Append")
// }