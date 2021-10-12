package kbucket

import (
	"bytes"
)

type ByPeer []PeerInfo

func (bp ByPeer) Len() int{
	return len(bp)
}

func (bp ByPeer) Swap(i, j int){

	bp[i], bp[j] = bp[j], bp[i]
}

//Non-Decreasing Order
func (bp ByPeer) Less(i, j int) bool {

	a := ConvertPeerID(bp[i].Id)
	b := ConvertPeerID(bp[j].Id)
	return bytes.Compare(a,b) < 0
}



