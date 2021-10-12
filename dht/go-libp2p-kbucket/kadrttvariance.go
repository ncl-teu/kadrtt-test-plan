package kbucket

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"math/big"
)


type VarianceInfo struct {
	id peer.ID
	variance *big.Int
}

type ByVariance []VarianceInfo

func (bp ByVariance) Len() int{
	return len(bp)
}

func (bp ByVariance) Swap(i, j int){

	bp[i], bp[j] = bp[j], bp[i]
}

//Non-Increasing Order of Variance
func (bp ByVariance) Less(i, j int) bool {
	a := bp[i].variance
	b := bp[j].variance
	return a.Cmp(b) > 0
}



