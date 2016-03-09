package main

type AppendEntriesRequestEvent struct {
	term              int
	leaderId          int
	prevLogIndex      int
	prevLogTerm       int
	data              []LogEntry
	leaderCommitIndex int
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
	if sm.currentTerm > msg.term {
		//request from invalid leader . Send leaders current term and failure
		action = append(action, Send{peerId: msg.leaderId, event: AppendEntriesResponseEvent{fromId: sm.myconfig.myId, term: sm.currentTerm, isSuccessful: false}})
	} else {
		//request from valid leader. Update the term, change to follower state and then process RPC
		if sm.currentTerm < msg.term {
			sm.votedFor = 0
		}
		sm.currentTerm = msg.term
		sm.state = "FOLLOWER"
		// call follower function
		action = sm.AppendEntriesRequestFollower(msg)
	}
	return action
}

func (sm *StateMachine) AppendEntriesRequestFollower(msg AppendEntriesRequestEvent) []interface{} {
	var action []interface{}
	if sm.currentTerm > msg.term {
		//request from invalid leader - send leader’s current term and failure
		action = append(action, Send{peerId: msg.leaderId, event: AppendEntriesResponseEvent{fromId: sm.myconfig.myId, term: sm.currentTerm, isSuccessful: false}})

	} else {

		if sm.currentTerm < msg.term {
			sm.votedFor = 0
		}
		sm.currentTerm = msg.term
		//Reset Election Timer
		action = append(action, Alarm{t: 200})
		action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
		//Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm

		if (msg.prevLogIndex != -1) && (len(sm.log) < msg.prevLogIndex || sm.log[msg.prevLogIndex].term != msg.prevLogTerm) {
			action = append(action, Send{peerId: msg.leaderId, event: AppendEntriesResponseEvent{fromId: sm.myconfig.myId, term: sm.currentTerm, isSuccessful: false}})
		} else {
			//Delete all entries starting from prevLogIndex+1 to logCurrentIndex and insert data from prevLogIndex+1
			sm.log = sm.log[:msg.prevLogIndex+1]
			sm.log = append(sm.log, msg.data...)
			//generate logstore action
			for i := msg.prevLogIndex + 1; i < msg.prevLogIndex+1+len(msg.data); i++ {
				action = append(action, LogStore{index: i, logData: msg.data[i-msg.prevLogIndex-1]})
			}
			//Update logCurrentIndex
			sm.logCurrentIndex = len(sm.log) - 1
			action = append(action, Send{peerId: msg.leaderId, event: AppendEntriesResponseEvent{fromId: sm.myconfig.myId, term: sm.currentTerm, isSuccessful: true}})
			if msg.leaderCommitIndex > sm.logCommitIndex {
				sm.logCommitIndex = Min(msg.leaderCommitIndex, sm.logCurrentIndex)
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
