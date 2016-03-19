package main

import "fmt"

type AppendEntriesRequestEvent struct {
	Term              int
	LeaderId          int
	PrevLogIndex      int
	PrevLogTerm       int
	Data              []LogEntry
	LeaderCommitIndex int
}

func (sm *StateMachine) AppendEntriesRequest(msg AppendEntriesRequestEvent) []interface{} {
	var action []interface{}
	switch sm.state {
	case "LEADER":
		action = sm.AppendEntriesRequestLeaderorCandidate(msg)
	case "FOLLOWER":
		action = sm.AppendEntriesRequestFollower(msg)
	case "CANDIDATE":
		action = sm.AppendEntriesRequestLeaderorCandidate(msg)
	}
	return action
}

func (sm *StateMachine) AppendEntriesRequestLeaderorCandidate(msg AppendEntriesRequestEvent) []interface{} {
	var action []interface{}
	if sm.currentTerm > msg.Term {
		//request from invalid leader . Send leaders current Term and failure
		action = append(action, Send{msg.LeaderId, AppendEntriesResponseEvent{FromId: sm.myconfig.myId, Term: sm.currentTerm, IsSuccessful: false, Index:sm.logCurrentIndex}})
	} else {
		//request from valid leader. Update the Term, change to follower state and then process RPC
		if sm.currentTerm < msg.Term {
			sm.votedFor = 0
		}
		sm.currentTerm = msg.Term
		sm.state = "FOLLOWER"
		// call follower function
		action = sm.AppendEntriesRequestFollower(msg)
	}
	return action
}

func (sm *StateMachine) AppendEntriesRequestFollower(msg AppendEntriesRequestEvent) []interface{} {
	var action []interface{}
	if sm.currentTerm > msg.Term {
		//request from invalid leader - send leader’s current Term and failure
		action = append(action, Send{PeerId: msg.LeaderId, Event: AppendEntriesResponseEvent{FromId: sm.myconfig.myId, Term: sm.currentTerm, IsSuccessful: false, Index : sm.logCurrentIndex}})

	} else {

		if sm.currentTerm < msg.Term {
			sm.votedFor = 0
		}
		sm.currentTerm = msg.Term
		//Reset Election Timer
	
		action = append(action, Alarm{t:Random(sm.electionTO) })
		action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
		//Reply false if log doesn’t contain an entry at PrevLogIndex whose Term matches PrevLogTerm		
		
		l:=len(sm.log)
		if(l==0){ // no entry in state machine log
			l=-1
		}
		if (msg.PrevLogIndex != -1) && (l < msg.PrevLogIndex || sm.log[msg.PrevLogIndex].Term != msg.PrevLogTerm) {
			action = append(action, Send{PeerId: msg.LeaderId, Event: AppendEntriesResponseEvent{FromId: sm.myconfig.myId, Term: sm.currentTerm, IsSuccessful: false, Index: sm.logCurrentIndex}})
		} else {
		
			//Delete all entries starting from PrevLogIndex+1 to logCurrentIndex and insert Data from PrevLogIndex+1
			sm.log = sm.log[:msg.PrevLogIndex+1]
			sm.log = append(sm.log, msg.Data...)
			//generate logstore action
			for i := msg.PrevLogIndex + 1; i < msg.PrevLogIndex+1+len(msg.Data); i++ {
	//			fmt.Printf("ID: %v , LogStore--> Index:%v, LogData : %v\n",sm.myconfig.myId,i, msg.Data[i-msg.PrevLogIndex-1])
				action = append(action, LogStore{Index: i, LogData: msg.Data[i-msg.PrevLogIndex-1]})
			}
			//Update logCurrentIndex
			sm.logCurrentIndex = len(sm.log) - 1
			action = append(action, Send{PeerId: msg.LeaderId, Event: AppendEntriesResponseEvent{FromId: sm.myconfig.myId, Term: sm.currentTerm, IsSuccessful: true, Index: sm.logCurrentIndex}})
			if msg.LeaderCommitIndex > sm.logCommitIndex {
				for j := sm.logCommitIndex + 1; j <= Min(msg.LeaderCommitIndex, sm.logCurrentIndex); j++ {
					fmt.Printf("Id: %v Commit Index:%v\n", sm.myconfig.myId, j)
					action = append(action, Commit{Index: j, Data: sm.log[j].Cmd, Err: nil})
				}
				sm.logCommitIndex = Min(msg.LeaderCommitIndex, sm.logCurrentIndex)
			}
		}

	}
	return action
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
