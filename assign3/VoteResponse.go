package main

type VoteResponseEvent struct {
	term          int
	isVoteGranted bool
}

func (sm *StateMachine) VoteResponse(msg VoteResponseEvent) []interface{} {
	var action []interface{}
	switch sm.state {
	case "LEADER":
		action = sm.VoteResponseLeader(msg)
	case "FOLLOWER":
		action = sm.VoteResponseFollower(msg)
	case "CANDIDATE":
		action = sm.VoteResponseCandidate(msg)
	}
	return action
}
func (sm *StateMachine) VoteResponseLeader(msg VoteResponseEvent) []interface{} {
	var action []interface{}
	//reject vote response
	return action
}
func (sm *StateMachine) VoteResponseFollower(msg VoteResponseEvent) []interface{} {
	var action []interface{}
	if msg.term > sm.currentTerm {
		sm.currentTerm = msg.term
		sm.votedFor = 0
		action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})

	}
	return action
}
func (sm *StateMachine) VoteResponseCandidate(msg VoteResponseEvent) []interface{} {
	var action []interface{}
	if msg.isVoteGranted == true {
		sm.yesVotesNum += 1
		if sm.yesVotesNum >= (len(sm.myconfig.peer)/2)+1 {
			// Elect it as leader
			sm.state = "LEADER"
			//store the state
			action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
			for i := 0; i < len(sm.myconfig.peer); i++ {
				// Initialize nextIndex and matchIndex and send heartbeat messages
				sm.matchIndex[i] = 0
				sm.nextIndex[i] = sm.logCurrentIndex + 1
				action = append(action, Send{peerId: sm.myconfig.peer[i], event: AppendEntriesRequestEvent{term: sm.currentTerm, leaderId: sm.myconfig.myId, prevLogIndex: sm.logCurrentIndex, prevLogTerm: sm.log[sm.logCurrentIndex].term, data: nil, leaderCommitIndex: sm.logCommitIndex}})
			}
			//reset heartbeat timer
			action = append(action, Alarm{t: 200})
		}
	} else {
		if msg.term > sm.currentTerm {
			sm.currentTerm = msg.term
			sm.state = "FOLLOWER"
			sm.votedFor = 0
			action = append(action, Alarm{t: 200})
			//store the state
			action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
		} else {
			sm.noVotesNum += 1
			if sm.noVotesNum >= (len(sm.myconfig.peer)/2)+1 {
				sm.state = "FOLLOWER"
				action = append(action, Alarm{t: 200})
				//store the state
				action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
			}
		}

	}
	return action
}
