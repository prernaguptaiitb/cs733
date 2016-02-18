package main

type VoteResponseEvent struct{
	term int
	isVoteGranted bool
}
func (sm *StateMachine) VoteResponse (msg VoteResponseEvent) ([] interface{}){
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
func (sm *StateMachine) VoteResponseLeader (msg VoteResponseEvent) ([] interface{}){
	var action []interface{}
	return action
}
func (sm *StateMachine) VoteResponseFollower (msg VoteResponseEvent) ([] interface{}){
	var action []interface{}
	return action
}
func (sm *StateMachine) VoteResponseCandidate (msg VoteResponseEvent) ([] interface{}){
	var action []interface{}
	return action
}