package main

import (
	kb "github.com/libp2p/go-libp2p-kbucket"
)


func main(){
	//var rt, _ = kb.NewKadRTTRT(20, nil, 1, nil, 100, nil)
	//kb.NewKadRTTRT(20, nil, 1, nil, 100, nil)
	kb.NewRoutingTable(20, nil, 1, nil, 100, nil)


}

