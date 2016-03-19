package main

import (
	"errors"
	//	"fmt"
)

type AppendEvent struct {
	Data []byte
}

func (sm *StateMachine) Append(msg AppendEvent) []interface{} {
	var action []interface{}
	switch sm.state {
	case "LEADER":
		action = sm.AppendLeader(msg)
	case "FOLLOWER":
		action = sm.AppendFollowerorCandidate(msg)
	case "CANDIDATE":
		action = sm.AppendFollowerorCandidate(msg)
	}
	return action
}

func (sm *StateMachine) AppendLeader(msg AppendEvent) []interface{} {
	//	fmt.Printf( " Id  : %v Append Message Recieved \n ", sm.myconfig.myId )
	var action []interface{}
	// append Data to leaders local log
	sm.logCurrentIndex++
	temp := LogEntry{sm.currentTerm, msg.Data}
	sm.log = append(sm.log, temp)
	action = append(action, LogStore{sm.logCurrentIndex, temp})
	//send AppendEntriesRequest to all peers
	for i := 0; i < len(sm.myconfig.peer); i++ {
		if sm.nextIndex[i] == 0 {
			//set previous log index and term to be -1 and 0 . We assume
			action = append(action, Send{sm.myconfig.peer[i], AppendEntriesRequestEvent{sm.currentTerm, sm.myconfig.myId, sm.nextIndex[i] - 1, 0, sm.log[sm.nextIndex[i]:], sm.logCommitIndex}})
		} else {
			action = append(action, Send{sm.myconfig.peer[i], AppendEntriesRequestEvent{sm.currentTerm, sm.myconfig.myId, sm.nextIndex[i] - 1, sm.log[sm.nextIndex[i]-1].Term, sm.log[sm.nextIndex[i]:], sm.logCommitIndex}})
		}
	}
	return action
}

func (sm *StateMachine) AppendFollowerorCandidate(msg AppendEvent) []interface{} {
	var action []interface{}
	action = append(action, Commit{-1, msg.Data, errors.New("Error in committing. Not a leader. Redirect")})
	return action
}
