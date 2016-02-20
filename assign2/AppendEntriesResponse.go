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
	i:=0
	for i=0; i<len(sm.myconfig.peer); i++ { 
		if msg.fromId == sm.myconfig.peer[i]{
			break
		}
	}
	if msg.isSuccessful == false {
			//receiver rejected because it is more updated(has higher term)
			if sm.currentTerm < msg.term {
				sm.state = "FOLLOWER"
				sm.currentTerm = msg.term
				sm.votedFor = 0
				action = append(action, Alarm{200})
				action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
			}else{	
				// follower rejected because previous entries didn't match
				
				sm.nextIndex[i]-=1
				action = append(action,Send{peerId : msg.fromId ,event : AppendEntriesRequestEvent{term : sm.currentTerm, leaderId : sm.myconfig.myId ,prevLogIndex : sm.nextIndex[i]-1, prevLogTerm : sm.log[sm.nextIndex[i]-1].term, data : sm.log[sm.nextIndex[i]:], leaderCommitIndex : sm.logCommitIndex}})
			}
	}else {
		//successfully appended at the follower.Update matchIndex and nextIndex
		sm.nextIndex[i] = sm.logCurrentIndex+1
		sm.matchIndex[i] = sm.logCurrentIndex
	//	fromid := i
		maxIndex:=-1
		//check if we can commit something
		for i :=0; i<len(sm.myconfig.peer); i++ { 
			numYes:=1
			if(sm.matchIndex[i] > sm.logCommitIndex && sm.matchIndex[i]>maxIndex && sm.log[sm.matchIndex[i]].term == sm.currentTerm ){  
				for j:=0; j<len(sm.myconfig.peer); j++ { 
					if(sm.matchIndex[j] >= sm.matchIndex[i] ){
						numYes+=1
					}
				}
				if numYes >= (len(sm.myconfig.peer)/2 +1) {
					if sm.matchIndex[i] > maxIndex{
						maxIndex = sm.matchIndex[i]
					}
				}
			} 
		}
		// If yes, commit and sent commit msg to layer above 
		if maxIndex != -1 {
			for k := sm.logCommitIndex+1 ; k <= maxIndex ; k++ {
				action = append(action, Commit{index : k , data : sm.log[k].cmd, err : nil})
			}
			sm.logCommitIndex = maxIndex
		}
	}
	return action
}
func (sm *StateMachine) AppendEntriesResponseFollowerorCandidate (msg AppendEntriesResponseEvent)([]interface{}){
	var action [] interface{}
	//reject the response
	if msg.term > sm.currentTerm  {
		sm.currentTerm = msg.term
		sm.state="FOLLOWER"	
		sm.votedFor=0
		action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
	}
	return action

}


