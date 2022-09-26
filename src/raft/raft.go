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

import "time"
import "crypto/rand"
import "math/big"
import "strconv"
import "bytes"
import "../labgob"
// import "fmt"





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

type RaftState int

const (
	FOLLOWER = iota
	CANDIDATE
	LEADER
) 

type Entry struct {
	Index        int
	Term         int
	Command      interface{}
}



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
	applyCond     *sync.Cond

	appendCond    []*sync.Cond 

	state     RaftState

	electionTime  *time.Timer
	heartBeatTime *time.Timer

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





// about append vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
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
	// only valid when Success is false
	ConflictTermIdx int
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()
	if rf.currentTerm > args.Term {
		reply.Term = rf.currentTerm
		reply.Success = false
		return 
	}

	if rf.currentTerm <= args.Term {
		rf.votedFor = -1    
		rf.state = FOLLOWER
		rf.currentTerm = args.Term
		rf.refreshElectionTime()
	}


	if rf.checkAppendCondition(args.PrevLogIndex, args.PrevLogTerm) {
		reply.Term = rf.currentTerm
		reply.Success = false

		if len(rf.log) <= args.PrevLogIndex {
			reply.ConflictTermIdx = rf.log[len(rf.log) - 1].Index + 1
		} else {
			term := rf.log[args.PrevLogIndex].Term
			idx := args.PrevLogIndex - 1
			for rf.log[idx].Term == term {
				idx--
			}
			reply.ConflictTermIdx = idx + 1
		}
		return

	} else {
		DPrintf("{Node %v} AppendEntries true!!! rf.currentTerm = %v args.Term = %v, args.LeaderId = %v, args.PrevLogIndex = %v, args.PrevLogTerm = %v, len(args.Entries) %v\n", 
							rf.me, rf.currentTerm, args.Term, args.LeaderId, args.PrevLogIndex, args.PrevLogTerm, len(args.Entries))
	DPrintf("***************************************************************************************************\n")
		rf.log = append(rf.log[0 : args.PrevLogIndex + 1], args.Entries...)
		reply.Term = rf.currentTerm
		reply.Success = true
	}
	

	if args.LeaderCommit > rf.commitIndex {
		rf.followerCommit(args.LeaderCommit)
	}

}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) checkAppendCondition(prevLogIndex int, prevLogTerm int) bool {
	return len(rf.log) <= prevLogIndex || rf.log[prevLogIndex].Term != prevLogTerm
}

func (rf *Raft) append2peer(peer int) {
	rf.mu.Lock()
	DPrintf("{Node %v} append2peer %v  rf.state = %v rf.currentTerm = %v, rf.commitIndex = %v len(rf.log) = %v\n", rf.me, peer, rf.state, rf.currentTerm, rf.commitIndex, len(rf.log))
	DPrintf("***************************************************************************************************\n")
	args := rf.makeAppendEntriesArgs(rf.nextIndex[peer])
	if rf.state != LEADER {
		rf.mu.Unlock()
		return
	}
	rf.mu.Unlock()
	reply := new(AppendEntriesReply)
	

	for !rf.killed() {
		if rf.sendAppendEntries(peer, &args, reply) {
			rf.mu.Lock()
			if rf.state != LEADER {
				rf.mu.Unlock()
				return
			}

			

			if args.Term < rf.currentTerm {
				rf.mu.Unlock()
				return
			}

			

			if reply.Term > rf.currentTerm {
				rf.currentTerm = reply.Term
				rf.state = FOLLOWER
				rf.votedFor = -1
				rf.persist()
				rf.mu.Unlock()
				return
			}

			if reply.Success {
				// args.Entries!!!
				if len(args.Entries) > 0 {
					rf.nextIndex[peer] = args.Entries[len(args.Entries) - 1].Index + 1
					rf.matchIndex[peer] = args.Entries[len(args.Entries) - 1].Index
				}
				// leaders can only commit logs with term same as his own
				if len(args.Entries) > 0 && args.Entries[len(args.Entries) - 1].Term == rf.currentTerm {
					rf.leaderCommit()
				}
				rf.mu.Unlock()
				return
			} else {
				// skip over conflicting log with same term
				rf.nextIndex[peer] = reply.ConflictTermIdx
				args = rf.makeAppendEntriesArgs(rf.nextIndex[peer])
				reply = new(AppendEntriesReply)
				rf.mu.Unlock()
			}
		}
	}
}

func (rf *Raft) makeAppendEntriesArgs(nextIndex int) AppendEntriesArgs {
	var l []Entry
	lastIndex, _ := rf.getlastIndexAndTerm()
	if lastIndex < nextIndex - 1 {
		DPrintf("panic!!!!!***************************************************************************************************\n")
	} else if lastIndex < nextIndex {
		l = make([]Entry, 0)
		
	} else {
		l = append([]Entry{}, rf.log[nextIndex:]...)   
	}

	return AppendEntriesArgs {
		Term: rf.currentTerm,
		LeaderId: rf.me,
		PrevLogIndex: nextIndex - 1,
		PrevLogTerm: rf.log[nextIndex - 1].Term,
		Entries: l,
		LeaderCommit: rf.commitIndex,
	}
}

func (rf *Raft) getlastIndexAndTerm() (int, int) {
	return rf.log[len(rf.log) - 1].Index, rf.log[len(rf.log) - 1].Term
}

func (rf *Raft) boardcast(ifHeartBeat bool) {
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}	
		if ifHeartBeat {
			go rf.append2peer(peer)
		} else {
			rf.appendCond[peer].Signal()
		}
	}
}

func (rf *Raft) refreshHeartBeatTime() {
	if rf.heartBeatTime == nil {
		rf.heartBeatTime = time.NewTimer(125 * time.Millisecond)
	} else {
		rf.heartBeatTime.Reset(125 * time.Millisecond)
	}
}
// vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv








// about election =============================================================================
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

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()

	// if candidate's term is smaller than mine, reject
	if args.Term < rf.currentTerm {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		return
	}

	// if rpc request or response contains term T > currentTerm, set curremtTerm to T and convert to FOLLOWERS
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.state = FOLLOWER
		rf.votedFor = -1
		rf.refreshElectionTime()
	}

	// if votedFor == -1 or CandidateId，and the candidate's log is at least as new as mine, vote for the candidate
	if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && rf.checkVoteCondition(args.LastLogIndex, args.LastLogTerm) {
		DPrintf("{Node %v} agree!!!  rf.state = %v rf.currentTerm = %v\n", rf.me, rf.state, rf.currentTerm)
		DPrintf("***************************************************************************************************\n")
		reply.Term, reply.VoteGranted = rf.currentTerm, true
		rf.refreshElectionTime()
		rf.votedFor = args.CandidateId
	} else {
		DPrintf("{Node %v} reject!!!  rf.state = %v rf.currentTerm = %v args.Term %v args.LastLogIndex %v, args.LastLogTerm %v\n", rf.me, rf.state, rf.currentTerm, args.Term, args.LastLogIndex, args.LastLogTerm)
		DPrintf("***************************************************************************************************\n")
		reply.Term, reply.VoteGranted = rf.currentTerm, false
	}

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
		args := rf.makeRequestVoteArgs()
		reply := new(RequestVoteReply)

		//---------------------------

		go func (peer int) {
			// receive rpc reply
			if rf.sendRequestVote(peer, &args, reply) {
				rf.mu.Lock()
				defer rf.mu.Unlock()
				// here we should check that rf.currentTerm hasn't changed since the decision to become a candidate.
				if rf.state == CANDIDATE && reply.Term == rf.currentTerm && args.Term == rf.currentTerm {
					if reply.VoteGranted {
						votecnt += 1
						if votecnt >= len(rf.peers) / 2 {
							rf.convert2Leader()
							rf.boardcast(true)
							rf.refreshHeartBeatTime()
							rf.persist()
						}
					}
				}
				// if rpc request or response contains term T > currentTerm, set curremtTerm to T and convert to FOLLOWERS
				if reply.Term > rf.currentTerm {
					rf.currentTerm = reply.Term
					rf.votedFor = -1
					rf.state = FOLLOWER
					rf.persist()
				}
			}
		}(peer)
	}
}

func (rf *Raft) makeRequestVoteArgs() RequestVoteArgs {
	lastIndex, lastTerm := rf.getlastIndexAndTerm()
	return RequestVoteArgs {
		Term: rf.currentTerm,
		CandidateId: rf.me,
		LastLogIndex: lastIndex,
		LastLogTerm: lastTerm,
	}
}

func (rf *Raft) convert2Leader() {
	rf.state = LEADER
	lastIndex, _ := rf.getlastIndexAndTerm()
	for peer := range rf.peers {
		rf.nextIndex[peer] = lastIndex + 1
		rf.matchIndex[peer] = 0
	}
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

func (rf *Raft) checkVoteCondition(lastLogIndex int, lastLogTerm int) bool {
	lastIndex, lastTerm := rf.getlastIndexAndTerm()
	return lastTerm < lastLogTerm || (lastTerm == lastLogTerm && lastIndex <= lastLogIndex)
}
// =============================================================================================







// about commit --------------------------------------------------------------------------------
func (rf *Raft) leaderCommit() {
	start := rf.commitIndex
	end := rf.log[len(rf.log) - 1].Index
	for start < end {
		cnt := 0
		for peer := range rf.peers {
			if rf.matchIndex[peer] >= start + 1 {
				cnt += 1
			} 
		}

		if cnt > len(rf.peers) / 2 {
			start++
		} else {
			break
		}
	}

	if rf.commitIndex < start {
		
		DPrintf("{Node %v} currentTerm %v leaderCommit rf.commitIndex %v to start %v\n", rf.me, rf.currentTerm, rf.commitIndex, start)
		DPrintf("***************************************************************************************************\n")
		rf.commitIndex = start
		rf.applyCond.Signal()
	}
	
}

func (rf *Raft) followerCommit(leaderCommit int) {
	rf.commitIndex = leaderCommit
	// signal
	rf.applyCond.Signal()
}
// ---------------------------------------------------------------------------------------------






// goruntime xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
func (rf *Raft) timer() {
	for !rf.killed() {
		select {
		case <- rf.heartBeatTime.C:
			rf.mu.Lock()
			if rf.state == LEADER {
				rf.boardcast(true)
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

func (rf *Raft) applyLogs() {
	for !rf.killed() {
		rf.mu.Lock()
		for rf.lastApplied >= rf.commitIndex {
			rf.applyCond.Wait()
		}
		
		
		lastApplied, commitIndex := rf.lastApplied, rf.commitIndex
		slice := make([]Entry, commitIndex - lastApplied)
		copy(slice, rf.log[lastApplied + 1: commitIndex + 1])
		
		rf.mu.Unlock()

		for _, entry := range slice {
			rf.applyCh <- ApplyMsg {
				CommandValid: true,
				Command: entry.Command,
				CommandIndex: entry.Index,
			}
		}

		rf.mu.Lock()
		if rf.lastApplied < commitIndex {
			DPrintf("{Node %v} currentTerm %v applyLogs, rf.lastApplied %v, commitIndex %v\n", rf.me, rf.currentTerm, rf.lastApplied, commitIndex)
			DPrintf("***************************************************************************************************\n")
			rf.lastApplied = commitIndex
		}
		rf.mu.Unlock()
	}
	DPrintf("{Node %v} currentTerm %v is killed!!!\n", rf.me, rf.currentTerm)
	DPrintf("***************************************************************************************************\n")
}

func (rf *Raft) appendAndWait(peer int) {
	rf.appendCond[peer].L.Lock()
	defer rf.appendCond[peer].L.Unlock()
	for !rf.killed() {
		for !rf.needAppend(peer) {
			rf.appendCond[peer].Wait()
		}
		rf.append2peer(peer)
	}
	
}

func (rf *Raft) needAppend(peer int) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.state == LEADER && rf.matchIndex[peer] < len(rf.log) - 1
}
// xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx









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
	rf.mu.Lock()
	defer rf.mu.Unlock()

	isLeader := rf.state == LEADER
	if !isLeader {
		return -1, -1, false
	}

	index := len(rf.log)
	term := rf.currentTerm
	

	// Your code here (2B).
	
	entry := Entry {
		Index: index,
		Term: term,
		Command: command,
	}

	rf.log = append(rf.log, entry)
	rf.nextIndex[rf.me] = index + 1
	rf.matchIndex[rf.me] = index

	rf.boardcast(false)

	DPrintf("{Node %v} receive client start, currentTerm %v, index %v, term %v, isLeader %v, command %v\n", rf.me, rf.currentTerm, index, term, isLeader, command)
	DPrintf("***************************************************************************************************\n")
	rf.persist()
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
	rf.applyCond = sync.NewCond(&rf.mu)

	rf.state = FOLLOWER
	rf.refreshElectionTime()
	rf.refreshHeartBeatTime()

	rf.currentTerm = 0   
	rf.votedFor = -1      
	rf.log = make([]Entry, 1)   
	
	rf.log[0] = Entry {
		Index: 0,
		Term: 0,
	}


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
	go rf.applyLogs()

	rf.appendCond = make([]*sync.Cond, len(peers))

	for peer := range rf.peers {
		rf.matchIndex[peer], rf.nextIndex[peer] = 0, len(rf.log)
		if peer == rf.me {
			continue
		}	
		rf.appendCond[peer] = sync.NewCond(&sync.Mutex{})
		go rf.appendAndWait(peer)
	}

	return rf
}






