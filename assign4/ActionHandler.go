package main

import (
	"github.com/cs733-iitb/cluster"
	"time"
)

func (rn *RaftNode) AlarmHandler(ac Alarm) {
	rn.timer.Reset(time.Duration(ac.t) * time.Millisecond)
}

func (rn *RaftNode) SendHandler(ac Send) {
	var ev interface{}
	switch ac.Event.(type) {
	case VoteRequestEvent:
		ev = ac.Event.(VoteRequestEvent)
	case VoteResponseEvent:
		ev = ac.Event.(VoteResponseEvent)
	case AppendEntriesRequestEvent:
		ev = ac.Event.(AppendEntriesRequestEvent)
	case AppendEntriesResponseEvent:
		ev = ac.Event.(AppendEntriesResponseEvent)
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
	lgLastIndex := rn.logFP.GetLastIndex()
	if int(lgLastIndex) >= ac.Index {
		rn.logFP.TruncateToEnd(int64(ac.Index))
	} else {
		err := rn.logFP.Append(ac.LogData)
		assert(err == nil)
	}

}

func (rn *RaftNode) StateStoreHandler(ac StateStore) {
	WriteState(rn.stateFP, ac)
}
