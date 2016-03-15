package main

import (
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/log"
	"time"
)

func (rn *RaftNode) AlarmHandler(ac Alarm) {
	val := rn.timeoutVal
	time.Sleep(time.Duration(ac.t) * time.Millisecond)
	if val == rn.timeoutVal {
		(rn.TimeoutCh) <- TimeoutEvent{}
	}
}

func (rn *RaftNode) SendHandler(ac Send) {
	var ev interface{}
	switch ac.event.(type) {
	case VoteRequestEvent:
		ev = ac.event.(VoteRequestEvent)
	case VoteResponseEvent:
		ev = ac.event.(VoteResponseEvent)
	case AppendEntriesRequestEvent:
		ev = ac.event.(AppendEntriesRequestEvent)
	case AppendEntriesResponseEvent:
		ev = ac.event.(AppendEntriesResponseEvent)
	}
	rn.srvr.Outbox() <- &cluster.Envelope{Pid: ac.peerId, Msg: ev}
}

func (rn *RaftNode) CommitHandler(ac Commit) {
	var ci CommitInfo
	ci.Data = ac.data
	ci.Index = ac.index
	ci.Err = ac.err
	rn.CommitCh <- ci
}

func (rn *RaftNode) LogStoreHandler(ac LogStore) {
	lgFile := rn.rc.LogDir + "/" + "logfile"
	lg, err := log.Open(lgFile)
	defer lg.Close()
	lgLastIndex := lg.GetLastIndex()
	if int(lgLastIndex) >= ac.index {
		lg.TruncateToEnd(int64(ac.index))
	} else {
		err = lg.Append(ac.logData)
		assert(err == nil)
	}

}

func (rn *RaftNode) StateStoreHandler(ac StateStore) {
	stateFile := rn.rc.StateDir + "/" + "mystate"
	state, err := log.Open(stateFile)
	defer state.Close()
	assert(err == nil)
	state.TruncateToEnd(0)
	err = state.Append(ac.state)
	assert(err == nil)
	err = state.Append(ac.term)
	assert(err == nil)
	err = state.Append(ac.votedFor)
	assert(err == nil)
}

