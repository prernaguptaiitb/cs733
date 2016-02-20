package main

import(
	"fmt"
	"testing"
	"reflect"
)

// check if state machine in expected state
func expectStateMachine(t *testing.T, responsesm StateMachine , expectedsm StateMachine, errstr string){
	ok := true
	if responsesm.state != expectedsm.state {
			ok = false
			errstr += fmt.Sprintf("State mismatch")
	}
	if responsesm.currentTerm != expectedsm.currentTerm {
			ok = false
			errstr += fmt.Sprintf("Term mismatch")
	}
	if responsesm.votedFor != expectedsm.votedFor {
			ok = false
			errstr += fmt.Sprintf("VotedFor mismatch")
	}
	if !reflect.DeepEqual(responsesm.log, expectedsm.log) {
			ok = false
			errstr += fmt.Sprintf("Log mismatch")
	}
	if responsesm.logCurrentIndex != expectedsm.logCurrentIndex {
			ok = false
			errstr += fmt.Sprintf("logCurrentIndex mismatch")
	}
	if responsesm.logCommitIndex != expectedsm.logCommitIndex {
			ok = false
			errstr += fmt.Sprintf("logCommitIndex mismatch")
	}
	if !reflect.DeepEqual(responsesm.nextIndex, expectedsm.nextIndex) {
			ok = false
			errstr += fmt.Sprintf("NextIndex mismatch")
	}
	if !reflect.DeepEqual(responsesm.matchIndex, expectedsm.matchIndex) {
			ok = false
			errstr += fmt.Sprintf("MatchIndex mismatch")
	}
	if responsesm.yesVotesNum != expectedsm.yesVotesNum {
			ok = false
			errstr += fmt.Sprintf("yesVotesNum mismatch")
	}
	if responsesm.noVotesNum != expectedsm.noVotesNum {
			ok = false
			errstr += fmt.Sprintf("noVotesNum mismatch")
	}
	if !ok {
		t.Fatal(errstr)
	}
}

// check if actions are returned if expected
func expectAction(t *testing.T, response []interface{} , expected []interface{}, errstr string) {
	ok := true
	if len(response) != len(expected){
		ok = false
		errstr += fmt.Sprintf("No of response actions not equal to expected")
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
   		errstr += fmt.Sprintf("Response action not as expected")
   	}
	flag := false
	for _, expval := range expected{
		for _,resval := range response{
			if reflect.DeepEqual(expval, resval){
				flag=true
				break
			}
		}
		if !flag {
			ok = false
			errstr += fmt.Sprintf("Expected %v action not generated", expval)
		}
	}

	if !ok {
		t.Fatal(fmt.Sprintf(errstr))
	}
}

func TestFollowerTimeout (t *testing.T){
	config := Config{1, []int{2,3,4,5}}
	mylog := []LogEntry{{1,[]byte("read")},{2,[]byte("cas")}}
	//nextIn := []int{2,2,2,2}
	//matchIn := []int{0,0,0,0}
	//sm := StateMachine{config,"FOLLOWER",3, 0,log,1,0,nextIn,matchIn,0,0}
	sm := StateMachine{myconfig : config, state : "FOLLOWER",currentTerm : 3, votedFor : 2,log : mylog,yesVotesNum:0}
	_=sm.ProcessEvent(TimeoutEvent{})
	expectStateMachine(t,sm,StateMachine{state : "CANDIDATE", currentTerm : 4, votedFor :1, log : mylog ,yesVotesNum:1 },"Error in follower timeout")
}

