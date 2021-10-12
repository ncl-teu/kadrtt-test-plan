package kbucket

import (
	"container/list"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// A helper struct to sort peers by their distance to the local node
type peerRTTDistance struct {
	p        peer.ID
	distance ID
	rtt      time.Duration
}

// peerDistanceSorter implements sort.Interface to sort peers by xor distance
type peerRTTDistanceSorter struct {
	peers  []peerRTTDistance
	target ID
}

func (pds *peerRTTDistanceSorter) Len() int { return len(pds.peers) }
func (pds *peerRTTDistanceSorter) Swap(a, b int) {
	pds.peers[a], pds.peers[b] = pds.peers[b], pds.peers[a]
}
func (pds *peerRTTDistanceSorter) Less(a, b int) bool {
	//return pds.peers[a].distance.less(pds.peers[b].distance)
	return pds.peers[a].rtt <= pds.peers[b].rtt
}

// Append the peer.ID to the sorter's slice. It may no longer be sorted.
func (pds *peerRTTDistanceSorter) appendPeer(p peer.ID, pDhtId ID, rtt time.Duration) {
	pds.peers = append(pds.peers, peerRTTDistance{
		p:        p,
		distance: xor(pds.target, pDhtId),
		rtt:      rtt,
	})
}

// Append the peer.ID values in the list to the sorter's slice. It may no longer be sorted.
func (pds *peerRTTDistanceSorter) appendPeersFromList(l *list.List) {
	for e := l.Front(); e != nil; e = e.Next() {
		pds.appendPeer(e.Value.(*PeerInfo).Id, e.Value.(*PeerInfo).dhtId, e.Value.(*PeerInfo).rtt)
	}
}

func (pds *peerRTTDistanceSorter) sort() {
	sort.Sort(pds)
}

// Sort the given peers by their ascending distance from the target. A new slice is returned.
/*func SortPeersByRTT(peers []peer.ID, target ID) []peer.ID {
	sorter := peerRTTDistanceSorter{
		peers:  make([]peerRTTDistance, 0, len(peers)),
		target: target,
	}
	for _, p := range peers {
		sorter.appendPeer(p, ConvertPeerID(p), p.)
	}
	sorter.sort()
	out := make([]peer.ID, 0, sorter.Len())
	for _, p := range sorter.peers {
		out = append(out, p.p)
	}
	return out
}*/
