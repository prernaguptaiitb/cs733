package main

import (
	"encoding/gob"
	"fmt"
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

// data goes in via Append, comes out as CommitInfo from the node's CommitChannel
// Index is valid only if err == nil
type CommitInfo struct {
	Data  []byte
	Index int
	Err   error // Err can be errred
}

type RaftNode struct {
	// implements Node interface
	rc       RaftConfig
	sm       StateMachine
	srvr     cluster.Server   // server object for communication
	EventCh  chan interface{} // for any event other than timeout
	CommitCh chan CommitInfo  //for committed entries
	timer    *time.Timer
	quit     chan bool
	stateFP  *os.File // pointer to the state file
	logFP    *log.Log
}

type NetConfig struct {
	Id   int
	Host string
	Port int
}

type RaftConfig struct {
	cluster          []NetConfig // Information about all servers, including this.
	Id               int         // this node's id.
	LogDir           string      // Log file directory for this node
	StateFile        string      // State file directory for this node. State file stores the state, current term and voted For in this order.
	ElectionTimeout  int
	HeartbeatTimeout int
}

type SMState struct {
	State       string
	CurrentTerm int
	VotedFor    int
}

func New(RaftNode_config RaftConfig) RaftNode {
	//make raftnode object and set it
	var rn RaftNode
	rn.rc = RaftNode_config
	rn.InitializeStateMachine(RaftNode_config)
	rn.EventCh = make(chan interface{}, 50000)
	rn.CommitCh = make(chan CommitInfo, 50000)
	rand.Seed(time.Now().UnixNano())
	rn.timer = time.NewTimer(time.Duration(Random(RaftNode_config.ElectionTimeout)) * time.Millisecond)

	rn.quit = make(chan bool)
	rn.srvr, _ = cluster.New(RaftNode_config.Id, "Config.json") //make server object for communication
	// register events
	gob.Register(VoteRequestEvent{})
	gob.Register(VoteResponseEvent{})
	gob.Register(AppendEntriesRequestEvent{})
	gob.Register(AppendEntriesResponseEvent{})

	return rn
}

func (rn *RaftNode) InitializeStateMachine(RaftNode_config RaftConfig) {

	//	var smobj StateMachine

	// initialize config structure of state machine
	rn.sm.myconfig.myId = RaftNode_config.Id
	for i := 0; i < len(RaftNode_config.cluster); i++ {
		if RaftNode_config.cluster[i].Id != RaftNode_config.Id {
			rn.sm.myconfig.peer = append(rn.sm.myconfig.peer, RaftNode_config.cluster[i].Id)
		}
	}

	// initialize state, current term and voted For of the state machine
	stateFile := RaftNode_config.StateFile
	rn.stateFP = OpenFile(stateFile)
	data := ReadState(rn.stateFP)
	rn.sm.state = data.State
	rn.sm.currentTerm = data.CurrentTerm
	rn.sm.votedFor = data.VotedFor

	// initialize statemachine log
	lgFile := RaftNode_config.LogDir + "/" + "logfile"
	var err error
	rn.logFP, err = log.Open(lgFile)
	rn.logFP.RegisterSampleEntry(LogEntry{})
	assert(err == nil)
	i := int(rn.logFP.GetLastIndex())
	for cnt := 0; cnt <= i; cnt++ {
		data, err := rn.logFP.Get(int64(cnt))
		assert(err == nil)
		lgentry, ok := data.(LogEntry)
		assert(ok)
		rn.sm.log = append(rn.sm.log, lgentry)
	}
	rn.sm.logCurrentIndex = int(rn.logFP.GetLastIndex())
	rn.sm.logCommitIndex = -1

	//initialize match index and next index
	for i := 0; i < len(RaftNode_config.cluster); i++ {
		if RaftNode_config.cluster[i].Id != RaftNode_config.Id {
			rn.sm.nextIndex = append(rn.sm.nextIndex, rn.sm.logCurrentIndex+1)
			rn.sm.matchIndex = append(rn.sm.matchIndex, -1)

		}
	}

	rn.sm.yesVotesNum = 0
	rn.sm.noVotesNum = 0
	rn.sm.electionTO = RaftNode_config.ElectionTimeout
	rn.sm.heartbeatTO = RaftNode_config.HeartbeatTimeout
	//	return smobj
}

// Client's message to Raft node
func (rn *RaftNode) Append(data []byte) {
	rn.EventCh <- AppendEvent{Data: data}
}

// A channel for client to listen on. What goes into Append must come out of here at some point
func (rn *RaftNode) CommitChannel() <-chan CommitInfo {
	return rn.CommitCh
}

// Last known committed index in the log. This could be -1 until the system stabilizes.
func (rn *RaftNode) CommittedIndex() int {
	return rn.sm.logCommitIndex
}

// Returns the data at a log index, or an error.
func (rn *RaftNode) Get(index int) (error, []byte) {
	lgentry, err := rn.logFP.Get(int64(index))
	le, _ := lgentry.(LogEntry)
	data := le.Cmd
	return err, data
}

// Node's id
func (rn *RaftNode) Id() int {
	return rn.rc.Id
}

// Id of leader. -1 if unknown
func (rn *RaftNode) LeaderId() int {
	return rn.sm.votedFor
}

// Signal to shut down all goroutines, stop sockets, flush log and close it, cancel timers.
func (rn *RaftNode) Shutdown() {
	rn.stateFP.Close()
	rn.logFP.Close()
	rn.quit <- true
	rn.srvr.Close()

}

func InitializeState(sd string) {
	f := OpenFile(sd)
	WriteState(f, StateStore{State: "FOLLOWER", Term: 0, VotedFor: 0})
	CloseFile(f)
}

func RestartNode(i int, clusterconf []NetConfig) RaftNode {

	ld := "myLogDir" + strconv.Itoa(i)
	sd := "StateId_" + strconv.Itoa(i)
	eo := 4000
	rc := RaftConfig{cluster: clusterconf, Id: i, LogDir: ld, StateFile: sd, ElectionTimeout: eo, HeartbeatTimeout: 600}
	rs := New(rc)
	return rs

}

func BringNodeUp(i int, clusterconf []NetConfig) RaftNode {
	sd := "StateId_" + strconv.Itoa(i)
	InitializeState(sd)
	rs := RestartNode(i, clusterconf)
	return rs
}

func (rn *RaftNode) processEvents() {
	var actions []interface{}
	for {
		var ev interface{}
		select {
		case ev = <-rn.EventCh:
			actions = rn.sm.ProcessEvent(ev)
			rn.doActions(actions)
		case <-rn.timer.C:
			actions = rn.sm.ProcessEvent(TimeoutEvent{})
			rn.doActions(actions)
		case env := <-rn.srvr.Inbox():
			switch env.Msg.(type) {
			case VoteRequestEvent:
				ev = env.Msg.(VoteRequestEvent)
			case VoteResponseEvent:
				ev = env.Msg.(VoteResponseEvent)
			case AppendEntriesRequestEvent:
				ev = env.Msg.(AppendEntriesRequestEvent)
			case AppendEntriesResponseEvent:
				ev = env.Msg.(AppendEntriesResponseEvent)

			}
			rn.EventCh <- ev
		case <-rn.quit:
			return
		}
	}
}

func (rn *RaftNode) doActions(actions []interface{}) {
	var ac interface{}
	for i := 0; i < len(actions); i++ {
		ac = actions[i]
		switch ac.(type) {
		case Alarm:
			res := ac.(Alarm)
			rn.AlarmHandler(res)
		case Send:
			res := ac.(Send)
			rn.SendHandler(res)
		case Commit:
			res := ac.(Commit)
			rn.CommitHandler(res)
		case LogStore:
			res := ac.(LogStore)
			rn.LogStoreHandler(res)
		case StateStore:
			res := ac.(StateStore)
			rn.StateStoreHandler(res)
		}
	}
}

func assert(val bool) {
	if !val {
		fmt.Println("Assertion Failed")
		os.Exit(1)
	}

}
func rmlog(lgFile string) {
	os.RemoveAll(lgFile)
}

func PrintVoteReqEvent(ev VoteRequestEvent) {
	fmt.Printf("Term : %v, CandidateId : %v, LastLogIndex :%v , LastLogTerm : %v\n", ev.Term, ev.CandidateId, ev.LastLogIndex, ev.LastLogTerm)
}

func PrintVoteResEvent(ev VoteResponseEvent) {
	fmt.Printf("Term : %v, IsVoteGranted : %v\n", ev.Term, ev.IsVoteGranted)
}

func PrintAppendEntriesReqEvent(ev AppendEntriesRequestEvent) {
	fmt.Printf("Term : %v, LeaderId : %v, PrevLogIndex : %v, PrevLogTerm: %v, Data : %v, LeaderCommitIndex : %v\n", ev.Term, ev.LeaderId, ev.PrevLogIndex, ev.PrevLogTerm, ev.Data, ev.LeaderCommitIndex)
}

func PrintAppendEntriesResEvent(ev AppendEntriesResponseEvent) {
	fmt.Printf("FromID : %v, Term : %v , Index : %v , IsSuccessful : %v\n ", ev.FromId, ev.Term, ev.Index, ev.IsSuccessful)
}
