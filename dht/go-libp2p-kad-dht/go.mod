module github.com/libp2p/go-libp2p-kad-dht

go 1.14

require (
	github.com/gogo/protobuf v1.3.2
	github.com/google/gopacket v1.1.19
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-detect-race v0.0.1
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-ipns v0.1.2
	github.com/ipfs/go-log v1.0.5
	github.com/jbenet/goprocess v0.1.4
	github.com/libp2p/go-eventbus v0.2.1
	github.com/libp2p/go-libp2p v0.14.4
	github.com/libp2p/go-libp2p-core v0.8.6
	//github.com/libp2p/go-libp2p-kad-dht v0.13.1
	github.com/libp2p/go-libp2p-kbucket v0.4.7
	//github.com/ncl-teu/go-libp2p-kadrtt-kbucket v0.0.0
	//github.com/ncl-teu/go-libp2p-kadrtt-dht v0.0.0
	github.com/libp2p/go-libp2p-peerstore v0.2.8
	github.com/libp2p/go-libp2p-record v0.1.3
	github.com/libp2p/go-libp2p-routing-helpers v0.2.3
	github.com/libp2p/go-libp2p-swarm v0.5.3
	github.com/libp2p/go-libp2p-testing v0.4.2
	github.com/libp2p/go-msgio v0.0.6
	github.com/libp2p/go-netroute v0.1.6
	github.com/multiformats/go-base32 v0.0.3
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/multiformats/go-multibase v0.0.3
	github.com/multiformats/go-multihash v0.0.15
	github.com/stretchr/testify v1.7.0
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1
	go.opencensus.io v0.23.0
	go.uber.org/zap v1.18.1
)

replace github.com/libp2p/go-libp2p-kbucket => ../go-libp2p-kbucket

//require github.com/ncl-teu/go-libp2p-kadrtt-kbucket v0.3.0
//replace github.com/libp2p/go-libp2p-kbucket => github.com/ncl-teu/go-libp2p-kadrtt-kbucket
//replace github.com/libp2p/go-libp2p-kad-dht => /home/kanemih/testground/libp2p-test-plans/dht/go-libp2p-kad-dht

//replace github.com/libp2p/go-libp2p-kbucket => /plan/go-libp2p-kbucket
