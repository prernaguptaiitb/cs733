package main

import (
	//	"errors"
	"fmt"
	"encoding/json"
	"github.com/cs733-iitb/log"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"
	"runtime"
)

type Peer struct {
	Id      int
	Address string
}

type ClusterConfig struct {
	Peers []Peer
}

func readJSONFile() ClusterConfig {
	content, err := ioutil.ReadFile("Config.json")
	if err != nil {
		fmt.Print("Error:", err)
	}
	var conf ClusterConfig
	err = json.Unmarshal(content, &conf)
	if err != nil {
		fmt.Print("Error:", err)
	}
	return conf
}

// This func creates a state file for storing initila state of each raft machine
func initialState(sd string) {
	stateFile := sd + "/" + "mystate"
//	rmlog(stateFile)
	st, err := log.Open(stateFile)
	//st.SetCacheSize(50)
	assert(err == nil)
	st.RegisterSampleEntry(SMState{})
	defer st.Close()
	err = st.Append(SMState{State: "FOLLOWER", CurrentTerm: 0, VotedFor: 0})
	assert(err == nil)
}

func clearFiles(file string){
	rmlog(file)
}

func makeRafts(conf ClusterConfig) []RaftNode {
	clusterconf := make([]NetConfig, len(conf.Peers))
	rafts := make([]RaftNode, len(conf.Peers))
	for j := 0; j < len(conf.Peers); j++ {
		clusterconf[j].Id = conf.Peers[j].Id
		clusterconf[j].Host = strings.Split(conf.Peers[j].Address, ":")[0]
		clusterconf[j].Port, _ = strconv.Atoi(strings.Split(conf.Peers[j].Address, ":")[1])
	}

	//	clusterconfig:=[]NetConfig{{Id:1, Host:"localhost", Port:8001},{Id:2, Host:"localhost", Port:8002},{Id:3, Host:"localhost", Port:8003},{Id:4, Host:"localhost", Port:8004},{Id:5, Host:"localhost", Port:8005}}
	var ld string
	var sd string
	for i := 1; i <= len(conf.Peers); i++ {

		ld = "myLogDir" + strconv.Itoa(i)
		clearFiles(ld+"/logfile")
		sd = "myStateDir" + strconv.Itoa(i)
		clearFiles(sd+"/mystate")
		initialState(sd)
		eo:= 1000+10*i
		rc := RaftConfig{cluster: clusterconf, Id: i, LogDir: ld, StateDir: sd, ElectionTimeout: eo, HeartbeatTimeout: 100}
		rafts[i-1] = New(rc)
		go rafts[i-1].processEvents()
	}

	return rafts
}

func getLeaderID(rafts []RaftNode) int {
	//find the number of votes of each node. If any node has got majority of votes then it is leader. If majority has not voted for a single node then election failure. Return -1 in this case
	majority := (len(rafts) / 2) + 1
	votes := make(map[int]int)
	for i := 0; i < len(rafts); i++ {
		votedFor := rafts[i].LeaderId()
		votes[votedFor] += 1
	}
	for key, value := range votes {
		if value >= majority {
			return key
		}
	}
	return 0
}

func TestBasic(t *testing.T) {
	runtime.GOMAXPROCS(1010)
	conf := readJSONFile()
	rafts := makeRafts(conf)
//	time.Sleep( 100 * time.Millisecond)
	leaderId := 0 // leader Id = 0 indicates election failure
	for leaderId == 0 {
		
		leaderId= getLeaderID(rafts)
	}
	time.Sleep( 10 * time.Millisecond)
//	fmt.Printf("Leader id : %v\n", leaderId)
	leader := rafts[leaderId-1]
	leader.Append([]byte("read"))
	time.Sleep( 60 * time.Second) 
	for _, node:= range rafts {
		select { // to avoid blocking on channel.
			case ci := <- node.CommitChannel():
				if ci.Err != nil {t.Fatal(ci.Err)}
				if string(ci.Data) != "read" {
					t.Fatal("Got different data")
				}
			default: t.Fatal("Expected message on all nodes")
		}
	}
}

func TestMultipleAppends (t *testing.T){

}


