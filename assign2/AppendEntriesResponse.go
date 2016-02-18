package main

type AppendEntriesResponseEvent struct{
	fromId int
	term int
	isSuccessful bool
}
func (sm *StateMachine) AppendEntriesResponse (msg AppendEntriesResponseEvent) ([] interface{}){
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
func (sm *StateMachine) AppendEntriesResponseLeader (msg AppendEntriesResponseEvent) ([] interface{}){
	var action [] interface{}
	if msg.isSuccessful == false {
			//receiver rejected because it is more updated(has higher term)
			if sm.currentTerm < msg.term {
				sm.state = "FOLLOWER"
				sm.currentTerm = msg.term
			}else{	
				// follower rejected because previous entries didn't match
				sm.nextIndex[msg.fromId]-=1
				action = append(action,Send{peerId : msg.fromId ,event : AppendEntriesRequestEvent{term : sm.currentTerm, leaderId : sm.myconfig.myId ,prevLogIndex : sm.nextIndex[msg.fromId]-1, prevLogTerm : sm.log[sm.nextIndex[msg.fromId]-1].term, data : sm.log[sm.nextIndex[msg.fromId]:], leaderCommitIndex : sm.logCommitIndex}})
			}
	}else {
		//successfully appended at the follower
		sm.nextIndex[msg.fromId] = sm.logCurrentIndex+1
		sm.matchIndex[msg.fromId] = sm.logCurrentIndex
		//update matchIndex and send commit 

	}
	return action
}
func (sm *StateMachine) AppendEntriesResponseFollowerorCandidate (msg AppendEntriesResponseEvent){
	//reject the response
	if msg.term > sm.currentTerm  {
		sm.currentTerm = msg.term	
	}
}


