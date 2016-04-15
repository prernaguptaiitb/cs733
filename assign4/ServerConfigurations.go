package main

import(
	"strconv"
	"strings"
)

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



