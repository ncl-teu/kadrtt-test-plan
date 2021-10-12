package main

import (
	"fmt"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/test-plans/dht/test"
	"time"
)

var testCasesRtt = map[string]interface{}{
	"find-peers":        test.FindPeers,
	"find-providers":    test.FindProviders,
	"provide-stress":    test.ProvideStress,
	"store-get-value":   test.StoreGetValue,
	"get-closest-peers": test.GetClosestPeers,
	"bootstrap-network": test.BootstrapNetwork,
	"all":               test.All,
}


func main() {
	t1 := time.Now()
		//var rt, _ = kb.NewKadRTTRT(20, nil, 1, nil, 100, nil)
		kb.NewRoutingTable(20, nil, 1, nil, 100, nil)
		//dht.NewDHT()
	//run.InvokeMap(testCasesRtt)
   t2 := time.Now()


   fmt.Print(t2.Sub(t1).Nanoseconds())


}
