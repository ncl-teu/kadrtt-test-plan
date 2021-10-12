module github.com/libp2p/test-plans/dht

go 1.14

require (
	github.com/gogo/protobuf v1.3.2
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-leveldb v0.4.2
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-ipns v0.1.2
	github.com/libp2p/go-libp2p v0.14.4
	github.com/libp2p/go-libp2p-asn-util v0.0.0-20201026210036-4f868c957324 // indirect
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.8.6
	github.com/libp2p/go-libp2p-kad-dht v0.11.1
	github.com/libp2p/go-libp2p-kbucket v0.4.7
	github.com/libp2p/go-libp2p-swarm v0.5.3
	github.com/libp2p/go-libp2p-transport-upgrader v0.4.6
	github.com/libp2p/go-libp2p-xor v0.0.0-20200501025846-71e284145d58
	github.com/libp2p/go-tcp-transport v0.2.7
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/multiformats/go-multiaddr-net v0.2.0
	github.com/pkg/errors v0.9.1
	github.com/testground/sdk-go v0.2.7
	go.uber.org/zap v1.18.1
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
)

//replace (
//github.com/libp2p/go-libp2p-kad-dht v0.11.1 => github.com/ncl-teu/go-libp2p-kad-dht v0.0.0-20211011063324-ed0e586ce6f4
//github.com/libp2p/go-libp2p-kbucket v0.4.7 => github.com/ncl-teu/go-libp2p-kbucket v0.0.0-20211011135728-979a873b3a6e
//)

replace github.com/libp2p/go-libp2p-kad-dht => ./go-libp2p-kad-dht

replace github.com/libp2p/go-libp2p-kbucket => ./go-libp2p-kbucket
