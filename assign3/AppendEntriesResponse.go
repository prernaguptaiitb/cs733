package main

type AppendEntriesResponseEvent struct {
	FromId       int
	Term         int
	IsSuccessful bool
}

func (sm *StateMachine) AppendEntriesResponse(msg AppendEntriesResponseEvent) []interface{} {
	var action []interface{}
	switch sm.state {
	case "LEADER":
		action = sm.AppendEntriesResponseLeader(msg)
	case "FOLLOWER":
		sm.AppendEntriesResponseFollowerorCandidate(msg)
	case "CANDIDATE":
		sm.AppendEntriesResponseFollowerorCandidate(msg)
	}
	return action
}
func (sm *StateMachine) AppendEntriesResponseLeader(msg AppendEntriesResponseEvent) []interface{} {
	var action []interface{}
	i := 0
	for i = 0; i < len(sm.myconfig.peer); i++ {
		if msg.FromId == sm.myconfig.peer[i] {
			break
		}
	}
	if msg.IsSuccessful == false {
		//receiver rejected because it is more updated(has higher Term)
		if sm.currentTerm < msg.Term {
			sm.state = "FOLLOWER"
			sm.currentTerm = msg.Term
			sm.votedFor = 0
			action = append(action, Alarm{t: Random(sm.electionTO)})
			action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
		} else {
			// follower rejected because previous entries didn't match
			sm.nextIndex[i] = Max(0, sm.nextIndex[i]-1)
			var prevlogterm int
			if sm.nextIndex[i]==0{
				prevlogterm = 0
			}else{
				prevlogterm = sm.log[sm.nextIndex[i]-1].Term
			}
			action = append(action, Send{PeerId: msg.FromId, Event: AppendEntriesRequestEvent{Term: sm.currentTerm, LeaderId: sm.myconfig.myId, PrevLogIndex: sm.nextIndex[i] - 1, PrevLogTerm: prevlogterm, Data: sm.log[sm.nextIndex[i]:], LeaderCommitIndex: sm.logCommitIndex}})
		}
	} else {
		//successfully appended at the follower.Update matchIndex and nextIndex
		sm.nextIndex[i] = sm.logCurrentIndex + 1
		sm.matchIndex[i] = sm.logCurrentIndex
		//	FromId := i
		maxIndex := -1
		//check if we can commit something
		for i := 0; i < len(sm.myconfig.peer); i++ {
			numYes := 1
			if sm.matchIndex[i] > sm.logCommitIndex && sm.matchIndex[i] > maxIndex && sm.log[sm.matchIndex[i]].Term == sm.currentTerm {
				for j := 0; j < len(sm.myconfig.peer); j++ {
					if sm.matchIndex[j] >= sm.matchIndex[i] {
						numYes += 1
					}
				}
				if numYes >= (len(sm.myconfig.peer)/2 + 1) {
					if sm.matchIndex[i] > maxIndex {
						maxIndex = sm.matchIndex[i]
					}
				}
			}
		}
		// If yes, commit and sent commit msg to layer above
		if maxIndex != -1 {
			for k := sm.logCommitIndex + 1; k <= maxIndex; k++ {
				action = append(action, Commit{Index: k, Data: sm.log[k].Cmd, Err: nil})
			}
			sm.logCommitIndex = maxIndex
		}
	}
	return action
}
func (sm *StateMachine) AppendEntriesResponseFollowerorCandidate(msg AppendEntriesResponseEvent) []interface{} {
	var action []interface{}
	//reject the response
	if msg.Term > sm.currentTerm {
		sm.currentTerm = msg.Term
		sm.state = "FOLLOWER"
		sm.votedFor = 0
		action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
	}
	return action

}
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
