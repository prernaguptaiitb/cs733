package main

type VoteRequestEvent struct {
	term         int
	candidateId  int
	lastLogIndex int
	lastLogTerm  int
}

func (sm *StateMachine) VoteRequest(msg VoteRequestEvent) []interface{} {
	var action []interface{}
	switch sm.state {
	case "LEADER":
		action = sm.VoteRequestLeaderorCandidate(msg)
	case "FOLLOWER":
		action = sm.VoteRequestFollower(msg)
	case "CANDIDATE":
		action = sm.VoteRequestLeaderorCandidate(msg)
	}
	return action
}
func (sm *StateMachine) VoteRequestLeaderorCandidate(msg VoteRequestEvent) []interface{} {
	var action []interface{}
	if sm.currentTerm < msg.term {
		// Update the term and change to follower state
		sm.currentTerm = msg.term
		sm.state = "FOLLOWER"
		sm.votedFor = 0
		//if candidate log is at least as up-to-date as this leaders log, then grant vote otherwise not
		if (sm.log[sm.logCurrentIndex].term < msg.lastLogTerm) || ((sm.log[sm.logCurrentIndex].term == msg.lastLogTerm) && (sm.logCurrentIndex <= msg.lastLogIndex)) {
			//grant vote
			sm.votedFor = msg.candidateId

			action = append(action, Send{peerId: msg.candidateId, event: VoteResponseEvent{term: sm.currentTerm, isVoteGranted: true}})

		} else {
			// do not grant vote
			action = append(action, Send{peerId: msg.candidateId, event: VoteResponseEvent{term: sm.currentTerm, isVoteGranted: false}})
		}
		action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
		action = append(action, Alarm{t: Random(sm.electionTO)})
	} else {
		// reject vote
		action = append(action, Send{peerId: msg.candidateId, event: VoteResponseEvent{term: sm.currentTerm, isVoteGranted: false}})
	}
	return action
}

func (sm *StateMachine) VoteRequestFollower(msg VoteRequestEvent) []interface{} {
	var action []interface{}
	flag := false
	if (sm.currentTerm == msg.term && (sm.votedFor == 0 /*assuming no server has id=0 */ || sm.votedFor == msg.candidateId)) || (sm.currentTerm < msg.term) {
		if sm.currentTerm < msg.term {
			sm.currentTerm = msg.term
			sm.votedFor = 0
			flag = true
		}
		//if candidate log is at least as up-to-date as this leaders log, then grant vote otherwise not
		if (sm.log[sm.logCurrentIndex].term < msg.lastLogTerm) || ((sm.log[sm.logCurrentIndex].term == msg.lastLogTerm) && (sm.logCurrentIndex <= msg.lastLogIndex)) {
			// grant vote
			sm.votedFor = msg.candidateId
			if sm.currentTerm < msg.term {
				flag = true
			}
			action = append(action, Alarm{t: Random(sm.electionTO)})
			action = append(action, Send{peerId: msg.candidateId, event: VoteResponseEvent{term: sm.currentTerm, isVoteGranted: true}})
		} else {
			// do not grant vote
			action = append(action, Send{peerId: msg.candidateId, event: VoteResponseEvent{term: sm.currentTerm, isVoteGranted: false}})
		}
		if flag {
			action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
		}
	} else {
		// reject vote
		action = append(action, Send{peerId: msg.candidateId, event: VoteResponseEvent{term: sm.currentTerm, isVoteGranted: false}})

	}
	return action
}
