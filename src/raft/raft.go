package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import "sync"
import "sync/atomic"
import "../labrpc"

// import "bytes"
// import "../labgob"

// my add start
import "time"
import "crypto/rand"
import "math/big"
import "strconv"
import "bytes"
import "../labgob"
// import "fmt"

// my add end



//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

// my add start

type RaftState int

const (
	FOLLOWER = iota
	CANDIDATE
	LEADER
) 

type Entry struct {
	Index        int
	Term         int
}


// my add end

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.


	applyCh       chan ApplyMsg

	state     RaftState

	electionTime  *time.Timer
	heartBeatTime *time.Timer
	lastIndex int
	lastTerm  int

	// // persistant state on all servers, updated on storage before responding to rpcs
	currentTerm    int
	votedFor       int
	log            []Entry          

	// // volatile state on all servers
	commitIndex    int
	lastApplied    int

	// // volatile state on the leader, reinitialized after each election
	nextIndex      []int
	matchIndex     []int

}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).

	term = rf.currentTerm
	isleader = rf.state == LEADER

	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {

	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}


//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}

	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var log []Entry        
	if d.Decode(&currentTerm) != nil ||
	   d.Decode(&votedFor) != nil ||
	   d.Decode(&log) != nil {
	  return
	} else {
	  rf.currentTerm = currentTerm
	  rf.votedFor = votedFor
	  rf.log = log
	}

}


type AppendEntriesArgs struct {
	Term	        int
	LeaderId	    int
	PrevLogIndex    int	
	PrevLogTerm	    int
	Entries	        []Entry
	LeaderCommit	int
}

type AppendEntriesReply struct {
	Term            int
	Success         bool
}


func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("{Node %v} receive AppendEntries!!!  rf.currentTerm = %v args.Term = %v args.LeaderId = %v\n", rf.me, rf.currentTerm, args.Term, args.LeaderId)
	DPrintf("***************************************************************************************************\n")
	if rf.currentTerm > args.Term {
		reply.Term = rf.currentTerm
		reply.Success = false
		return 
	}
	if len(args.Entries) == 0 {
		if rf.currentTerm <= args.Term {
			rf.votedFor = -1    
			rf.state = FOLLOWER
			rf.currentTerm = args.Term
			DPrintf("{Node %v} AppendEntries change currentTerm %v to %v\n", rf.me, rf.currentTerm, args.Term)
			DPrintf("***************************************************************************************************\n")
			rf.persist()
		}
		rf.refreshElectionTime()
		reply.Term = rf.currentTerm
		reply.Success = true
	}
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}





type RequestVoteArgs struct {
	// Your data here (2A, 2B).

	Term            int 
	CandidateId     int 
	LastLogIndex    int 
	LastLogTerm     int 
}

type RequestVoteReply struct {
	// Your data here (2A).

	Term            int 
	VoteGranted     bool 
}




// example RequestVote RPC handler.

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	
	rf.mu.Lock()
	defer rf.mu.Unlock()
	DPrintf("{Node %v} receive RequestVote!!!  rf.currentTerm = %v args.Term = %v args.CandidateId = %v\n", rf.me, rf.currentTerm, args.Term, args.CandidateId)
	DPrintf("***************************************************************************************************\n")

	// if candidate's term is smaller than mine, reject
	if args.Term < rf.currentTerm {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		return
	}

	// if rpc request or response contains term T > currentTerm, set curremtTerm to T and convert to FOLLOWERS
	if args.Term > rf.currentTerm {
		DPrintf("{Node %v} RequestVote change currentTerm %v to %v\n", rf.me, rf.currentTerm, args.Term)
		DPrintf("***************************************************************************************************\n")
		rf.currentTerm = args.Term
		rf.state = FOLLOWER
		rf.votedFor = -1
		rf.refreshElectionTime()
		rf.persist()
	}

	// if votedFor == -1 or CandidateId，and the candidate's log is at least as new as mine, vote for the candidate
	if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && rf.checkVoteCondition(args.LastLogIndex, args.LastLogTerm) {
		DPrintf("{Node %v} agree!!!  rf.state = %v rf.currentTerm = %v\n", rf.me, rf.state, rf.currentTerm)
		DPrintf("***************************************************************************************************\n")
		reply.Term, reply.VoteGranted = rf.currentTerm, true
		rf.refreshElectionTime()
		rf.votedFor = args.CandidateId
	} else {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
	}

	// rf.mu.Lock()
	// defer rf.mu.Unlock()
	// defer rf.persist()
	

	// if args.Term < rf.currentTerm || (args.Term == rf.currentTerm && rf.votedFor != -1 && rf.votedFor != args.CandidateId) {
	// 	reply.Term, reply.VoteGranted = rf.currentTerm, false
	// 	return
	// }
	// if args.Term > rf.currentTerm {
	// 	DPrintf("{Node %v} RequestVote change currentTerm %v to %v\n", rf.me, rf.currentTerm, args.Term)
	// 	DPrintf("***************************************************************************************************\n")
	// 	rf.state = FOLLOWER
	// 	rf.currentTerm, rf.votedFor = args.Term, -1
	// }
	// if !rf.checkVoteCondition(args.LastLogIndex, args.LastLogTerm) {
	// 	reply.Term, reply.VoteGranted = rf.currentTerm, false
	// 	return
	// }
	// rf.votedFor = args.CandidateId
	// rf.refreshElectionTime()
	// reply.Term, reply.VoteGranted = rf.currentTerm, true
}




func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) KickOffElection() {
	votecnt := 0
	DPrintf("{Node %v} KickOffElection!!!  rf.state = %v rf.currentTerm = %v\n", rf.me, rf.state, rf.currentTerm)
	DPrintf("***************************************************************************************************\n")
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		//--------------------------
		// we let this block out of goruntime, see rules 4 and 5 http://nil.csail.mit.edu/6.824/2020/labs/raft-locking.txt
		args := RequestVoteArgs {
			Term: rf.currentTerm,
			CandidateId: rf.me,
			LastLogIndex: rf.lastIndex,
			LastLogTerm: rf.lastTerm,
		}
		var reply RequestVoteReply

		//---------------------------

		go func (peer int) {
			// receive rpc reply
			if rf.sendRequestVote(peer, &args, &reply) {
				rf.mu.Lock()
				defer rf.mu.Unlock()

				// here we should check that rf.currentTerm hasn't changed since the decision to become a candidate.
				if rf.state == CANDIDATE && reply.Term == rf.currentTerm {
					if reply.VoteGranted {
						votecnt += 1
						if votecnt >= len(rf.log) / 2 {
							rf.state = LEADER
							// if broadCast() will blocked ？ No, it creates goruntimes to do the wait and process the reply
							rf.broadCast()
							rf.refreshHeartBeatTime()
							rf.persist()
						}
					}
				}
				// if rpc request or response contains term T > currentTerm, set curremtTerm to T and convert to FOLLOWERS
				if reply.Term > rf.currentTerm {
					DPrintf("{Node %v}  KickOffElection change currentTerm %v to %v\n", rf.me, rf.currentTerm, reply.Term)
					DPrintf("***************************************************************************************************\n")
					rf.currentTerm = reply.Term
					rf.state = FOLLOWER
					rf.persist()
				}
			}
		}(peer)
	}
}


func (rf *Raft) broadCast() {
	DPrintf("{Node %v} broadCast!!!  rf.state = %v rf.currentTerm = %v\n", rf.me, rf.state, rf.currentTerm)
	DPrintf("***************************************************************************************************\n")
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}

		empty := make([]Entry, 0)
		args := AppendEntriesArgs {
			Term: rf.currentTerm,
			LeaderId: rf.me,
			PrevLogIndex: 0,
			PrevLogTerm: 0,
			Entries: empty,
			LeaderCommit: rf.commitIndex,
		}
		var reply AppendEntriesReply


		go func (peer int) {
			// receive rpc reply
			if rf.sendAppendEntries(peer, &args, &reply) {
				rf.mu.Lock()
				defer rf.mu.Unlock()

				if reply.Term > rf.currentTerm {
					DPrintf("{Node %v} broadCast change currentTerm %v to %v\n", rf.me, rf.currentTerm, reply.Term)
					DPrintf("***************************************************************************************************\n")
					rf.currentTerm = reply.Term
					rf.state = FOLLOWER
					rf.persist()
				}
			}
		}(peer)
	}
}

func (rf *Raft) timer() {
	for !rf.killed() {
		select {
		case <- rf.heartBeatTime.C:
			rf.mu.Lock()
			if rf.state == LEADER {
				rf.broadCast()
				rf.refreshHeartBeatTime()
			}
			rf.mu.Unlock()

		case <- rf.electionTime.C:
			rf.mu.Lock()
			if rf.state != LEADER {
				rf.state = CANDIDATE
				rf.currentTerm += 1
				DPrintf("{Node %v} after add currentTerm %v rf.state %v\n", rf.me, rf.currentTerm, rf.state)
				DPrintf("***************************************************************************************************\n")
				rf.votedFor = rf.me
				rf.persist()
				rf.KickOffElection()
				rf.refreshElectionTime()
			}
			rf.mu.Unlock()
		}
	}
	DPrintf("{Node %v} currentTerm %v is killed!!!\n", rf.me, rf.currentTerm)
	DPrintf("***************************************************************************************************\n")
}


//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).


	return index, term, isLeader
}


func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).

	rf.applyCh = applyCh
	
	rf.state = FOLLOWER
	rf.refreshElectionTime()
	rf.refreshHeartBeatTime()
	rf.lastIndex = 0
	rf.lastTerm = 0

	rf.currentTerm = 100   
	rf.votedFor = -1      
	rf.log = make([]Entry, 0)   

	rf.commitIndex = 0 
	rf.lastApplied = 0   

	rf.nextIndex = make([]int, len(peers))       
	rf.matchIndex = make([]int, len(peers))     

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	
	DPrintf("-------------------------------\n")
	DPrintf("{Node %v} is created!!! rf.state = %v rf.currentTerm = %v \n", rf.me, rf.state, rf.currentTerm)
	DPrintf("-------------------------------\n")
	go rf.timer()


	return rf
}




func (rf *Raft) refreshElectionTime() {
	n, _ := rand.Int(rand.Reader, big.NewInt(150))
	num := n.String()
	number, _ := strconv.Atoi(num)
	if rf.electionTime == nil {
		rf.electionTime = time.NewTimer(time.Duration(number + 600) * time.Millisecond)
	} else {
		rf.electionTime.Reset(time.Duration(number + 600) * time.Millisecond)
	}
}

func (rf *Raft) refreshHeartBeatTime() {
	if rf.heartBeatTime == nil {
		rf.heartBeatTime = time.NewTimer(125 * time.Millisecond)
	} else {
		rf.heartBeatTime.Reset(125 * time.Millisecond)
	}
}

func (rf *Raft) checkVoteCondition(lastLogIndex int, lastLogTerm int) bool {
	return rf.lastTerm < lastLogTerm || (rf.lastTerm == lastLogTerm && rf.lastIndex <= lastLogIndex)
}

