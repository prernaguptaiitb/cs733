package main

import (
//	"errors"
	"fmt"
	"strconv"
	"testing"
	"io/ioutil"
    "encoding/json"
    "strings"
    "time"
)

type Peer struct{
    Id int
    Address string
}

type ClusterConfig struct{
    Peers []Peer
}

func readJSONFile() ClusterConfig{
	content, err := ioutil.ReadFile("Config.json")
    if err!=nil{
        fmt.Print("Error:",err)
    }
    var conf ClusterConfig
    err=json.Unmarshal(content, &conf)
    if err!=nil{
        fmt.Print("Error:",err)
    }
    return conf
}


func makeRafts( conf ClusterConfig ) []RaftNode {
	clusterconf:= make([]NetConfig,len(conf.Peers))
	rafts:= make([]RaftNode,len(conf.Peers))
	for j:=0; j<len(conf.Peers); j++{
		clusterconf[j].Id=conf.Peers[j].Id
		clusterconf[j].Host=strings.Split(conf.Peers[j].Address,":")[0]
		clusterconf[j].Port,_=strconv.Atoi(strings.Split(conf.Peers[j].Address,":")[1])
	}

//	clusterconfig:=[]NetConfig{{Id:1, Host:"localhost", Port:8001},{Id:2, Host:"localhost", Port:8002},{Id:3, Host:"localhost", Port:8003},{Id:4, Host:"localhost", Port:8004},{Id:5, Host:"localhost", Port:8005}}
	var ld string
	var sd string
	for i:=1; i<=len(conf.Peers); i++ {
		ld="myLogDir"+strconv.Itoa(i)
		sd="myStateDir"+strconv.Itoa(i)
		rc := RaftConfig{cluster:clusterconf, Id: i, LogDir:ld, StateDir:sd, ElectionTimeout:150, HeartbeatTimeout:75}
		go func() {
			rafts[i-1] = New(rc)
		}()
		
	}
	return rafts
}

func getLeaderID(rafts []RaftNode) int{
	//find the number of votes of each node. If any node has got majority of votes then it is leader. If majority has not voted for a single node then election failure. Return -1 in this case
	nodesNo:=len(rafts)
	majority:=(nodesNo/2)+1
	votes := make(map[int]int)
	for i:=0; i<len(rafts); i++ {
		votedFor:=rafts[i].LeaderId()
		votes[votedFor]+=1
	}
	for key, value := range votes {
    	if value >= majority{
    		return key
    	}
	}
	return -1
}

func TestBasic (t *testing.T) {
	conf:=readJSONFile()	
	rafts:=makeRafts(conf)
	leaderId:=-1 // leader Id = -1 indicates election failure
	for leaderId == -1 {
		time.Sleep(1 * time.Second)
		leaderId=getLeaderID(rafts)
	}
	leader := rafts[leaderId-1] 
	leader.Append([]byte("read"))
	time.Sleep(1 * time.Second)
}   