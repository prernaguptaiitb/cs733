package main

import(
	"fmt"
	"testing"
	"reflect"
	"errors"
)

// check if state machine in expected state
func expectStateMachine(t *testing.T, responsesm StateMachine , expectedsm StateMachine, errstr string){
	ok := true
	if responsesm.state != expectedsm.state {
			ok = false
			errstr += fmt.Sprintf("State mismatch\n")
	}
	if responsesm.currentTerm != expectedsm.currentTerm {
			ok = false
			errstr += fmt.Sprintf("Term mismatch\n")
	}
	if responsesm.votedFor != expectedsm.votedFor {
			ok = false
			errstr += fmt.Sprintf("VotedFor mismatch\n")
	}
	if !reflect.DeepEqual(responsesm.log, expectedsm.log) {
			ok = false
			errstr += fmt.Sprintf("Log mismatch\n")
	}
	if responsesm.logCurrentIndex != expectedsm.logCurrentIndex {
			ok = false
			errstr += fmt.Sprintf("logCurrentIndex mismatch\n")
	}
	if responsesm.logCommitIndex != expectedsm.logCommitIndex {
			ok = false
			errstr += fmt.Sprintf("logCommitIndex mismatch %v %v\n",responsesm.logCommitIndex,expectedsm.logCommitIndex )
	}
	if !reflect.DeepEqual(responsesm.nextIndex, expectedsm.nextIndex) {
			ok = false
			errstr += fmt.Sprintf("NextIndex mismatch %v %v\n", responsesm.nextIndex, expectedsm.nextIndex)
	}
	if !reflect.DeepEqual(responsesm.matchIndex, expectedsm.matchIndex) {
			ok = false
			errstr += fmt.Sprintf("MatchIndex mismatch\n")
	}
	if responsesm.yesVotesNum != expectedsm.yesVotesNum {
			ok = false
			errstr += fmt.Sprintf("yesVotesNum mismatch\n")
	}
	if responsesm.noVotesNum != expectedsm.noVotesNum {
			ok = false
			errstr += fmt.Sprintf("noVotesNum mismatch\n")
	}
	if !ok {
		t.Fatal(errstr)
	}
}
//check if actions are equal
func checkEqual(actionArr1 []interface{}, actionArr2 []interface{})(bool){
	flag := false
	for _, expval := range actionArr1{
		for _,resval := range actionArr2{
			if reflect.DeepEqual(expval, resval){
				flag = true
				break
			}
		}
		if !flag {
			fmt.Printf("%v %v", expval)
			return flag
		}
		flag = false
	}
	return true
}

// check if actions are returned as expected
func expectAction(t *testing.T, response []interface{} , expected []interface{}, errstr string) {
	ok := true
	if len(response) != len(expected){
		ok = false
		errstr += fmt.Sprintf("No of response actions not equal to expected\n")
	}
	resMap := make(map[reflect.Type]int)
	expMap := make(map[reflect.Type]int)   
	for _,val := range response {        
            resMap[reflect.TypeOf(val)] += 1
        }
    for _,val := range expected{
    		expMap[reflect.TypeOf(val)] += 1
    }
   	if !reflect.DeepEqual(resMap, expMap){
   		//check if action are generated as expected
   		ok = false
   		errstr += fmt.Sprintf("Response action not as expected\n")
   	}
   	if !(checkEqual(expected,response) && checkEqual(response,expected)){
			ok = false
			errstr += fmt.Sprintf("Expected action not generated\n")
	}
	if !ok {
		t.Fatal(fmt.Sprintf(errstr))
	}
}

func TestLeaderAppend(t *testing.T){
	config := Config{1, []int{2,3,4,5}}
	mylog := []LogEntry{{1,[]byte("read")},{2,[]byte("cas")}}
	sm := StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor : 1, log : mylog, logCurrentIndex : 1 ,logCommitIndex : 0, nextIndex : []int{0,2,2,1}}
	action:=sm.ProcessEvent(AppendEvent{[]byte("write")})
	newlog := []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("write")}}
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor : 1, log : newlog ,logCurrentIndex : 2,logCommitIndex : 0, nextIndex : []int{0,2,2,1}},"Error in leader append\n")
	expectAction(t,action,[]interface{}{LogStore{index : 2 ,logData : LogEntry{3,[]byte("write")}}, Send{peerId : 2, event : AppendEntriesRequestEvent{ term : 3, leaderId : 1 , prevLogIndex : -1 , prevLogTerm : 0 , data :newlog, leaderCommitIndex : 0}},Send{peerId : 3, event : AppendEntriesRequestEvent{ term : 3, leaderId : 1 , prevLogIndex : 1 , prevLogTerm : 2 , data :[]LogEntry{{3,[]byte("write")}}, leaderCommitIndex : 0}},Send{peerId : 4, event : AppendEntriesRequestEvent{ term : 3, leaderId : 1 , prevLogIndex : 1 , prevLogTerm : 2 , data : []LogEntry{{3,[]byte("write")}}, leaderCommitIndex : 0}},Send{peerId : 5, event : AppendEntriesRequestEvent{ term : 3, leaderId : 1 , prevLogIndex : 0 , prevLogTerm : 1 , data : []LogEntry{{2,[]byte("cas")},{3,[]byte("write")}}, leaderCommitIndex : 0}}},"Error in leader append Actions")
}

func TestAppendFollowerorCandidate(t *testing.T){
	config := Config{1, []int{2,3,4,5}}
	sm := StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 3}
	action:=sm.ProcessEvent(AppendEvent{[]byte("write")})
	expectAction(t,action,[]interface{}{Commit{ -1, []byte("write") , errors.New("Error in committing. Not a leader")}},"Error in append of follower or candidate")
}

func TestFollowerAppendEntriesRequest(t *testing.T){

	//peer 2 with next index = 0 . Leader sends 3 entries. term of peer 2 less than leader
	config := Config{2, []int{1,3,4,5}}
	sm := StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 2, votedFor : 1, log : []LogEntry{}, logCurrentIndex :-1, logCommitIndex:-1 }
	newlog := []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("write")}}
	action:=sm.ProcessEvent(AppendEntriesRequestEvent{ term : 3, leaderId : 1 , prevLogIndex : -1 , prevLogTerm : 0 , data :newlog, leaderCommitIndex : 0})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 3, votedFor : 0, log: newlog, logCurrentIndex:2 , logCommitIndex : 0},"Error in FollowerAppendEntriesRequest-case1")
	expectAction(t, action,[]interface{}{Alarm{200},StateStore{"FOLLOWER",3,0},LogStore{0,LogEntry{1,[]byte("read")}},LogStore{1,LogEntry{2,[]byte("cas")}},LogStore{2,LogEntry{3,[]byte("write")}},Send{1,AppendEntriesResponseEvent{2,3,true}}},"Error in append entries request follower")

	// peer 3 with next index =2 . Leader send only 1 entry . Term of peer 3 greater than leader
	config = Config{3, []int{1,2,4,5}}
	sm = StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 4, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{4,[]byte("delete")}}, logCurrentIndex :2, logCommitIndex:1 }
	action=sm.ProcessEvent(AppendEntriesRequestEvent{ term : 3, leaderId : 1 , prevLogIndex : 1 , prevLogTerm : 2 , data : []LogEntry{{3,[]byte("write")}}, leaderCommitIndex : 0})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 4, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{4,[]byte("delete")}}, logCurrentIndex :2, logCommitIndex:1 }, "Error in FollowerAppendEntriesRequest-case2")
	expectAction(t, action,[]interface{}{Send{1,AppendEntriesResponseEvent{3,4,false}}},"Error in FollowerAppendEntriesRequest-case2")
	
	// peer 4 with next index =2 . Leader send only 1 entry and previous log index and previous  log term did not match
	config = Config{4, []int{1,2,4,5}}
	sm = StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 2, votedFor : 1, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("delete")}}, logCurrentIndex :2, logCommitIndex:1 }
	newlog = []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{4,[]byte("write")},{4,[]byte("delete")}}
	action=sm.ProcessEvent(AppendEntriesRequestEvent{ term : 4, leaderId : 1 , prevLogIndex : 2 , prevLogTerm : 4 , data : []LogEntry{{4,[]byte("delete")}}, leaderCommitIndex : 1})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 4, votedFor : 0, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("delete")}}, logCurrentIndex :2, logCommitIndex:1 }, "Error in FollowerAppendEntriesRequest-case2")
	expectAction(t, action,[]interface{}{Alarm{200},StateStore{"FOLLOWER",4,0},Send{1,AppendEntriesResponseEvent{4,4,false}}},"Error in FollowerAppendEntriesRequest-case3")
	
	// Now suppose previous log term match, then log should be overwritten
	action=sm.ProcessEvent(AppendEntriesRequestEvent{ term : 4, leaderId : 1 , prevLogIndex : 1 , prevLogTerm : 2 , data : []LogEntry{{4,[]byte("write")},{4,[]byte("delete")}}, leaderCommitIndex : 1})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 4, votedFor : 0, log : newlog, logCurrentIndex :3, logCommitIndex:1 }, "Error in FollowerAppendEntriesRequest-case2")
	expectAction(t, action,[]interface{}{Alarm{200},StateStore{"FOLLOWER",4,0},LogStore{2,LogEntry{4,[]byte("write")}},LogStore{3,LogEntry{4,[]byte("delete")}}, Send{1,AppendEntriesResponseEvent{4,4,true}}},"Error in FollowerAppendEntriesRequest-case4")

}
func TestLeaderorCandidateAppendEntriesRequest(t *testing.T){
	//request from invalid leader
	config := Config{1, []int{2,3,4,5}}
	sm := StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor : 1}
	action := sm.ProcessEvent(AppendEntriesRequestEvent{term : 2, leaderId : 2 , prevLogIndex : 1 , prevLogTerm : 2 , data : []LogEntry{{3,[]byte("write")}}, leaderCommitIndex : 0})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor : 1},"Error in TestLeaderorCandidateAppendEntriesRequest")
	expectAction(t,action,[]interface{}{Send{2, AppendEntriesResponseEvent{1,3,false}}},"Error in TestLeaderorCandidateAppendEntriesRequest - case1")

	//request from valid leader
	sm = StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor : 1, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")}}, logCurrentIndex : 1, logCommitIndex : 1}
	action = sm.ProcessEvent(AppendEntriesRequestEvent{term : 4, leaderId : 2 , prevLogIndex : 1 , prevLogTerm : 2 , data : []LogEntry{{4,[]byte("write")}}, leaderCommitIndex : 1})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 4, votedFor : 0, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{4,[]byte("write")}}, logCurrentIndex : 2, logCommitIndex : 1},"Error in TestLeaderorCandidateAppendEntriesRequest")
	expectAction(t, action,[]interface{}{Alarm{200},StateStore{"FOLLOWER",4,0},LogStore{2,LogEntry{4,[]byte("write")}},Send{2,AppendEntriesResponseEvent{1,4,true}}},"Error in FollowerAppendEntriesRequest-case4")
}

func TestLeaderAppendEntriesResponse(t *testing.T){
	config := Config{1, []int{2,3,4,5}}
	sm := StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor : 1, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("write")},{3,[]byte("delete")}}, nextIndex : []int{0,2,2,1}, logCurrentIndex : 3, logCommitIndex : 1 }

	// unsuccessful response from some other valid leader
	action := sm.ProcessEvent(AppendEntriesResponseEvent{2, 4, false})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 4, votedFor : 0, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("write")},{3,[]byte("delete")}}, nextIndex : []int{0,2,2,1}, logCurrentIndex : 3, logCommitIndex : 1 },"Error in TestLeaderAppendEntriesResponse - case 1")
	expectAction(t,action,[]interface{}{Alarm{200}, StateStore{"FOLLOWER", 4, 0}},"error in TestLeaderAppendEntriesResponse - case 1")

	// follower rejected because previous entries didn't match
	sm = StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor : 1, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("write")},{3,[]byte("delete")}}, nextIndex : []int{0,2,2,1}, logCurrentIndex : 3, logCommitIndex : 1 }
	action = sm.ProcessEvent(AppendEntriesResponseEvent{3, 3, false})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor : 1, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("write")},{3,[]byte("delete")}}, nextIndex : []int{0,1,2,1}, logCurrentIndex : 3, logCommitIndex : 1 },"Error in TestLeaderAppendEntriesResponse - case 2")
	expectAction(t,action,[]interface{}{Send{3,AppendEntriesRequestEvent{3,1,0,1,[]LogEntry{{2,[]byte("cas")},{3,[]byte("write")},{3,[]byte("delete")}},1 }}},"Error in TestLeaderAppendEntriesResponse - case2")

	// successfully updated at the follower
	sm = StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor:1, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("write")},{3,[]byte("delete")}}, nextIndex : []int{0,2,2,1}, matchIndex : []int{0,0,2,1}, logCurrentIndex : 3, logCommitIndex : 1 }
	action = sm.ProcessEvent(AppendEntriesResponseEvent{3, 3, true})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "LEADER",currentTerm : 3, votedFor : 1, log : []LogEntry{{1,[]byte("read")},{2,[]byte("cas")},{3,[]byte("write")},{3,[]byte("delete")}}, nextIndex : []int{0,4,2,1}, matchIndex : []int{0,3,2,1}, logCurrentIndex : 3, logCommitIndex : 2 },"Error in TestLeaderAppendEntriesResponse - case 3")
	expectAction(t,action,[]interface{}{Commit{index : 2 , data : []byte("write"), err : nil}},"Error in TestLeaderAppendEntriesResponse - case3")

}
func TestCandidateorFollowerAppendEntriesResponse(t *testing.T){
	config := Config{1, []int{2,3,4,5}}
	sm := StateMachine{myconfig : config, state : "CANDIDATE",currentTerm : 3, votedFor : 1}
	_= sm.ProcessEvent(AppendEntriesResponseEvent{3, 4, false})
	expectStateMachine(t,sm,StateMachine{myconfig : config, state : "FOLLOWER", currentTerm : 4, votedFor :0 },"Error in TestCandidateorFollowerAppendEntriesResponse")
	//expectAction(t,action,[]interface{}{StateStore{"FOLLOWER", 4, 0}},"Error in TestCandidateorFollowerAppendEntriesResponse")

}
func TestFollowerTimeout (t *testing.T){
	config := Config{1, []int{2,3,4,5}}
	mylog := []LogEntry{{1,[]byte("read")},{2,[]byte("cas")}}
	sm := StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 3, votedFor : 2,log : mylog,logCurrentIndex : 1,yesVotesNum:0}
	action:=sm.ProcessEvent(TimeoutEvent{})
	expectStateMachine(t,sm,StateMachine{state : "CANDIDATE", currentTerm : 4, votedFor :1, log : mylog ,logCurrentIndex : 1,yesVotesNum:1 },"Error in follower timeout\n")
	expectAction(t,action,[]interface{}{Alarm{100},Send{peerId : 2, event : VoteRequestEvent{term : 4, candidateId : 1, lastLogIndex : 1, lastLogTerm : 2 }},Send{peerId : 3, event : VoteRequestEvent{term : 4, candidateId : 1, lastLogIndex : 1, lastLogTerm : 2 }},Send{peerId : 4, event : VoteRequestEvent{term : 4, candidateId : 1, lastLogIndex : 1, lastLogTerm : 2 }},Send{peerId : 5, event : VoteRequestEvent{term : 4, candidateId : 1, lastLogIndex : 1, lastLogTerm : 2 }},StateStore{state : "CANDIDATE", term : 4, votedFor : 1}},"Error in follower Timeout\n ")
}

func TestLeaderTimeout (t *testing.T){

}
