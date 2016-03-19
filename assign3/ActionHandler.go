package main

import (
	"github.com/cs733-iitb/cluster"
	"github.com/cs733-iitb/log"
	"time"
	//	"fmt"
)

func (rn *RaftNode) AlarmHandler(ac Alarm) {
	rn.timer.Reset(time.Duration(ac.t) * time.Millisecond)
}

func (rn *RaftNode) SendHandler(ac Send) {
	var ev interface{}
	switch ac.Event.(type) {
	case VoteRequestEvent:
		//		fmt.Printf("%v Id VoteRequestEvent Send \n", rn.rc.Id)
		ev = ac.Event.(VoteRequestEvent)
		//		PrintVoteReqEvent(ev.(VoteRequestEvent))
	case VoteResponseEvent:
		ev = ac.Event.(VoteResponseEvent)
		//		fmt.Printf("%v Id VoteResponseEvent Send\n", rn.rc.Id)
		//		PrintVoteResEvent(ev.(VoteResponseEvent))
	case AppendEntriesRequestEvent:
		//		fmt.Printf("%v Id AppendEntriesRequestEvent  send\n", rn.rc.Id)
		ev = ac.Event.(AppendEntriesRequestEvent)
		//		PrintAppendEntriesReqEvent(ev.(AppendEntriesRequestEvent))
	case AppendEntriesResponseEvent:
		//		fmt.Printf("%v Id AppendEntriesResponseEvent Send\n", rn.rc.Id)
		ev = ac.Event.(AppendEntriesResponseEvent)
		//		PrintAppendEntriesResEvent(ev.(AppendEntriesResponseEvent))
	}
	rn.srvr.Outbox() <- &cluster.Envelope{Pid: ac.PeerId, Msg: ev}
}

func (rn *RaftNode) CommitHandler(ac Commit) {
	//	fmt.Printf("%v Commit generated\n", rn.rc.Id)
	var ci CommitInfo
	ci.Data = ac.Data
	ci.Index = ac.Index
	ci.Err = ac.Err
	rn.CommitCh <- ci
}

func (rn *RaftNode) LogStoreHandler(ac LogStore) {
	//	fmt.Printf("%v LogStore generated\n", rn.rc.Id)
	lgFile := rn.rc.LogDir + "/" + "logfile"
	lg, err := log.Open(lgFile)
	lg.RegisterSampleEntry(LogEntry{})
	defer lg.Close()
	lgLastIndex := lg.GetLastIndex()
	if int(lgLastIndex) >= ac.Index {
		lg.TruncateToEnd(int64(ac.Index))
	} else {
		err = lg.Append(ac.LogData)
		assert(err == nil)
	}

}

func (rn *RaftNode) StateStoreHandler(ac StateStore) {
	//	fmt.Printf("%v StateStore generated\n", rn.rc.Id)
	stateFile := rn.rc.StateDir + "/" + "mystate"
	state, err := log.Open(stateFile)
	defer state.Close()
	assert(err == nil)
	state.TruncateToEnd(0)
	ss := SMState{State: ac.State, CurrentTerm: ac.Term, VotedFor: ac.VotedFor}
	err = state.Append(ss)
	assert(err == nil)
}
