package main

import(
	"strconv"
	"strings"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"os"
)


type Peer struct {
	Id      int
	Address string
	FSAddress string
}

type ClusterConfig struct {
	Peers []Peer
}

func IfError(err error, msg string){
	if err != nil {
		fmt.Print("Error:", msg)
		os.Exit(1)
	}
}

func readJSONFile(configFile string) ClusterConfig {
	content, err := ioutil.ReadFile(configFile)
	IfError(err,"Error in Reading Json File")
	var conf ClusterConfig
	err = json.Unmarshal(content, &conf)
	IfError(err,"Error in Unmarshaling Json File")
	return conf
}


func makeRaftNetConfig(conf ClusterConfig) []NetConfig {
	clusterconf := make([]NetConfig, len(conf.Peers))

	for j := 0; j < len(conf.Peers); j++ {
		clusterconf[j].Id = conf.Peers[j].Id
		clusterconf[j].Host = strings.Split(conf.Peers[j].Address, ":")[0]
		clusterconf[j].Port, _ = strconv.Atoi(strings.Split(conf.Peers[j].Address, ":")[1])
	}
	return clusterconf
}

func makeFSNetConfig(conf ClusterConfig) []FSConfig {
	fsconf := make([]FSConfig, len(conf.Peers))

	for j := 0; j < len(conf.Peers); j++ {
		fsconf[j].Id = conf.Peers[j].Id
		fsconf[j].Address = conf.Peers[j].FSAddress
	}
	return fsconf
}



