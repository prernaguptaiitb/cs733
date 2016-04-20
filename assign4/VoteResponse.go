package main

import "fmt"

type VoteResponseEvent struct {
	Term          int
	IsVoteGranted bool
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
	if msg.Term > sm.currentTerm {
		sm.currentTerm = msg.Term
		sm.votedFor = 0
		action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})

	}
	return action
}
func (sm *StateMachine) VoteResponseCandidate(msg VoteResponseEvent) []interface{} {
	var action []interface{}
	if msg.Term < sm.currentTerm {
		return action
	}
	if msg.IsVoteGranted == true {
		sm.yesVotesNum += 1
		if sm.yesVotesNum >= (len(sm.myconfig.peer)/2)+1 {
			// Elect it as leader
			sm.state = "LEADER"
			fmt.Printf("Leader Elected: %v, term=%v \n", sm.myconfig.myId, sm.currentTerm)
			//store the state
			action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
			var lastlogTerm int
			for i := 0; i < len(sm.myconfig.peer); i++ {
				// Initialize nextIndex and matchIndex and send heartbeat messages
				sm.matchIndex[i] = -1
				sm.nextIndex[i] = sm.logCurrentIndex + 1
				if sm.logCurrentIndex == -1 {
					lastlogTerm = 0
				} else {
					lastlogTerm = sm.log[sm.logCurrentIndex].Term
				}
				action = append(action, Send{PeerId: sm.myconfig.peer[i], Event: AppendEntriesRequestEvent{Term: sm.currentTerm, LeaderId: sm.myconfig.myId, PrevLogIndex: sm.logCurrentIndex, PrevLogTerm: lastlogTerm, Data: nil, LeaderCommitIndex: sm.logCommitIndex}})
			}
			//reset heartbeat timer
			action = append(action, Alarm{t: sm.heartbeatTO})
		}
	} else {
		if msg.Term > sm.currentTerm {
			sm.currentTerm = msg.Term
			sm.state = "FOLLOWER"
			sm.votedFor = 0
			action = append(action, Alarm{t: Random(sm.electionTO)})
			//store the state
			action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
		} else {
			sm.noVotesNum += 1
			if sm.noVotesNum >= (len(sm.myconfig.peer)/2)+1 {
				sm.state = "FOLLOWER"
				action = append(action, Alarm{t: Random(sm.electionTO)})
				//store the state
				action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
			}
		}

	}
	return action
}
