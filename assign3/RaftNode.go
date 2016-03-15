package main

import (
	"encoding/gob"
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/log"
	"os"
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
	rc RaftConfig
	sm         StateMachine
	srvr       cluster.Server    // server object for communication
	EventCh    chan interface{}  // for any event other than timeout
	TimeoutCh  chan TimeoutEvent //for timeout events
	CommitCh   chan CommitInfo   //for committed entries
	timeoutVal int64
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
	StateDir         string      // State file directory for this node. State file stores the state, current term and voted For in this order.
	ElectionTimeout  int
	HeartbeatTimeout int
}

func New(RaftNode_config RaftConfig) RaftNode {

	//make raftnode object and set it
	var rn RaftNode
	rn.rc=RaftNode_config
	//rn.rc.LogDir = RaftNode_config.LogDir
	//rn.rc.StateDir = RaftNode_config.StateDir
	rn.sm = InitializeStateMachine(RaftNode_config)
	rn.EventCh = make(chan interface{}, 1000)
	rn.TimeoutCh = make(chan TimeoutEvent)
	rn.CommitCh = make(chan CommitInfo, 1000)
	rn.timeoutVal = 0

	rn.srvr, _ = cluster.New(RaftNode_config.Id, "Config.json") //make server object for communication
	// register events
	gob.Register(VoteRequestEvent{})
	gob.Register(VoteResponseEvent{})
	gob.Register(AppendEntriesRequestEvent{})
	gob.Register(AppendEntriesResponseEvent{})

	return rn
}

func InitializeStateMachine(RaftNode_config RaftConfig) StateMachine {
	var smobj StateMachine
	// initialize config structure of state machine
	smobj.myconfig.myId = RaftNode_config.Id
	for i := 0; i < len(RaftNode_config.cluster); i++ {
		if RaftNode_config.cluster[i].Id != RaftNode_config.Id {
			smobj.myconfig.peer = append(smobj.myconfig.peer, RaftNode_config.cluster[i].Id)
		}
	}
	// initialize state, current term and voted For of the state machine
	var ok bool
	stateFile := RaftNode_config.StateDir + "/" + "mystate"
	state, err := log.Open(stateFile)
	defer state.Close()
	assert(err == nil)
	res, err := state.Get(0)
	assert(err == nil)
	smobj.state, ok = res.(string)
	assert(ok)
	res, err = state.Get(1)
	assert(err == nil)
	smobj.currentTerm, ok = res.(int)
	assert(ok)
	res, err = state.Get(2)
	assert(err == nil)
	smobj.votedFor, ok = res.(int)
	assert(ok)

	// initialize statemachine log
	lgFile := RaftNode_config.LogDir + "/" + "logfile"
	lg, err := log.Open(lgFile)
	defer lg.Close()
	assert(err == nil)
	i := int(lg.GetLastIndex())
	for cnt := 0; cnt <= i; cnt++ {
		data, err := lg.Get(int64(cnt))
		assert(err == nil)
		lgentry, ok := data.(LogEntry)
		assert(ok)
		smobj.log = append(smobj.log, lgentry)
	}
	smobj.logCurrentIndex = int(lg.GetLastIndex())
	smobj.logCommitIndex = -1
	//initialize match index and next index
	for i := 0; i < len(RaftNode_config.cluster); i++ {
		if RaftNode_config.cluster[i].Id != RaftNode_config.Id {
			smobj.nextIndex = append(smobj.nextIndex, smobj.logCurrentIndex+1)
			smobj.matchIndex = append(smobj.matchIndex, -1)

		}
	}

	smobj.yesVotesNum = 0
	smobj.noVotesNum = 0
	smobj.electionTO = RaftNode_config.ElectionTimeout
	smobj.heartbeatTO = RaftNode_config.HeartbeatTimeout
	return smobj
}

// Client's message to Raft node
func (rn *RaftNode) Append(data []byte) {
	rn.EventCh <- AppendEvent{data: data}
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
func (rn *RaftNode) Get(index int) (error, []byte){
	lgFile := rn.rc.LogDir + "/" + "logfile"
	lg, err := log.Open(lgFile)
	defer lg.Close()
	assert(err == nil)
	lgentry, err := lg.Get(int64(index)) 
    le,_ := lgentry.(LogEntry) 
    data := le.cmd
    return err, data
}

// Node's id
func (rn *RaftNode) Id() (int){
	return rn.rc.Id
}

// Id of leader. -1 if unknown
func (rn *RaftNode) LeaderId() int{
	return rn.sm.votedFor
}
// Signal to shut down all goroutines, stop sockets, flush log and close it, cancel timers.
func (rn *RaftNode) Shutdown(){
	//rmlog(rn.rc.LogDir+"/logfile")
	//rmlog(rn.rc.StateDir+"/stateFile")

}

func (rn *RaftNode) processEvents() {
	var actions []interface{}
	for {
		var ev interface{}
		select {
		case ev = <-rn.EventCh:
			actions = rn.sm.ProcessEvent(ev)
			rn.doActions(actions)
		case ev = <-rn.TimeoutCh:
			actions = rn.sm.ProcessEvent(ev)
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
			rn.timeoutVal += 1
			go rn.AlarmHandler(res)
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
		panic("Assertion Failed")
	}
}
func rmlog(lgFile string) {
	os.RemoveAll(lgFile)
}

