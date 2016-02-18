package main

type VoteRequestEvent struct{
	term int
	candidateId int
	lastLogIndex int
	lastLogTerm int
}

func (sm *StateMachine) VoteRequest (msg VoteRequestEvent) ([] interface{}){
	var action []interface{}
	switch sm.state {
				case "LEADER":
					action = sm.VoteRequestLeader(msg)			
				case "FOLLOWER":
					action = sm.VoteRequestFollower(msg)
				case "CANDIDATE":
					action = sm.VoteRequestCandidate(msg)
			}
		return action
}
func (sm *StateMachine) VoteRequestLeader (msg VoteRequestEvent) ([] interface{}){

}
func (sm *StateMachine) VoteRequestFollower (msg VoteRequestEvent) ([] interface{}){

}
func (sm *StateMachine) VoteRequestCandidate (msg VoteRequestEvent) ([] interface{}){
	
}
