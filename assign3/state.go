package main

import (
	
	"math/rand"
	"time"
)

type Config struct {
	myId int   //persistent
	peer []int //persistent
}

type StateMachine struct {
	myconfig        Config
	state           string     //persistent
	currentTerm     int        //persistent
	votedFor        int        //persistent
	log             []LogEntry //persistent
	logCurrentIndex int
	logCommitIndex  int
	nextIndex       []int
	matchIndex      []int
	yesVotesNum     int
	noVotesNum      int
	electionTO      int
	heartbeatTO     int
}

type LogEntry struct {
	Term int
	Cmd  []byte
}

func (sm *StateMachine) ProcessEvent(ev interface{}) []interface{} {
	var action []interface{}
	switch ev.(type) {

	case AppendEvent:
		cmd := ev.(AppendEvent)
		action = sm.Append(cmd)

	case AppendEntriesRequestEvent:
		cmd := ev.(AppendEntriesRequestEvent)
		action = sm.AppendEntriesRequest(cmd)

	case AppendEntriesResponseEvent:
		cmd := ev.(AppendEntriesResponseEvent)
		action = sm.AppendEntriesResponse(cmd)

	case VoteRequestEvent:
		cmd := ev.(VoteRequestEvent)
		action = sm.VoteRequest(cmd)

	case VoteResponseEvent:
		cmd := ev.(VoteResponseEvent)
		action = sm.VoteResponse(cmd)

	case TimeoutEvent:
		cmd := ev.(TimeoutEvent)
		action = sm.Timeout(cmd)

	// other cases
	default:
		println("Unrecognized")
	}
	return action
}

func Random(min int) int {
	//max := 2 * min
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(min) + min
//	return min
}
