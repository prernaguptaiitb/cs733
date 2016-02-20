package main

type TimeoutEvent struct{

}

func (sm *StateMachine) Timeout (msg TimeoutEvent) ([] interface{}){
	var action []interface{}
	switch sm.state {
				case "LEADER":
					action = sm.TimeoutLeader(msg)			
				case "FOLLOWER":
					action = sm.TimeoutFollower(msg)
				case "CANDIDATE":
					action = sm.TimeoutCandidate(msg)
			}
		return action
}
func (sm *StateMachine) TimeoutLeader (msg TimeoutEvent) ([] interface{}){
	var action []interface{}
	//heartbeat timeout
	for i :=0; i<len(sm.myconfig.peer); i++ { 
		if sm.nextIndex[i] == 0{
			action = append(action,Send{peerId : sm.myconfig.peer[i], event : AppendEntriesRequestEvent{ term : sm.currentTerm, leaderId : sm.myconfig.myId, prevLogIndex : sm.nextIndex[i]-1, prevLogTerm : 0, data : sm.log[sm.nextIndex[i]:] , leaderCommitIndex : sm.logCommitIndex}})
		}else{
			action = append(action,Send{peerId : sm.myconfig.peer[i], event : AppendEntriesRequestEvent{ term : sm.currentTerm, leaderId : sm.myconfig.myId, prevLogIndex : sm.nextIndex[i]-1, prevLogTerm : sm.log[sm.nextIndex[i]-1].term, data : sm.log[sm.nextIndex[i]:] , leaderCommitIndex : sm.logCommitIndex}})
		}			
	}
	return action
}
func (sm *StateMachine) TimeoutFollower (msg TimeoutEvent) ([] interface{}){
	var action []interface{}
	//election timeout
	sm.currentTerm +=1
	sm.state="CANDIDATE"
	sm.votedFor = sm.myconfig.myId
	action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
	sm.yesVotesNum=1
	sm.noVotesNum=0
	//Reset election timer
	action = append(action, Alarm{t:100})
	for i :=0; i<len(sm.myconfig.peer); i++ { 
			action = append(action,Send{peerId : sm.myconfig.peer[i], event : VoteRequestEvent{term : sm.currentTerm, candidateId : sm.myconfig.myId, lastLogIndex : sm.logCurrentIndex, lastLogTerm : sm.log[sm.logCurrentIndex].term }} )
		}
	return action
}
func (sm *StateMachine) TimeoutCandidate (msg TimeoutEvent) ([] interface{}){
	var action []interface{}
	//election timeout
	sm.currentTerm +=1
	sm.votedFor = sm.myconfig.myId
	action = append(action, StateStore{sm.state, sm.currentTerm, sm.votedFor})
	sm.yesVotesNum=1
	sm.noVotesNum=0
	//Reset election timer
	action = append(action, Alarm{0})
	for i :=0; i<len(sm.myconfig.peer); i++ { 
			action = append(action,Send{peerId : sm.myconfig.peer[i], event : VoteRequestEvent{term : sm.currentTerm, candidateId : sm.myconfig.myId, lastLogIndex : sm.logCurrentIndex, lastLogTerm : sm.log[sm.logCurrentIndex].term }} )
		}
	return action
}
