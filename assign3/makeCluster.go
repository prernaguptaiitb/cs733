package main
import "github.com/cs733-iitb/cluster"

func MakeCluster(){
	srvr,_ := cluster.New(1, "Config.json")
}
