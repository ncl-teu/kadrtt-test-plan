// package kbucket implements a kademlia 'k-bucket' routing table.
package kbucket

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	u "github.com/ipfs/go-ipfs-util"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"math"
	"math/big"

	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-kbucket/peerdiversity"

	logging "github.com/ipfs/go-log"
)

var log = logging.Logger("table")

var ErrPeerRejectedHighLatency = errors.New("peer rejected; latency too high")
var ErrPeerRejectedNoCapacity = errors.New("peer rejected; insufficient capacity")


// RoutingTable defines the routing table.
type RoutingTable struct {
	// the routing table context
	ctx context.Context
	// function to cancel the RT context
	ctxCancel context.CancelFunc


	// ID of the local peer
	local ID

	// Blanket lock, refine later for better performance
	tabLock sync.RWMutex

	// latency metrics
	metrics peerstore.Metrics

	// Maximum acceptable latency for peers in this cluster
	maxLatency time.Duration

	// kBuckets define all the fingers to other nodes.
	buckets    []*bucket
	bucketsize int

	cplRefreshLk   sync.RWMutex
	cplRefreshedAt map[uint]time.Time

	// notification functions
	PeerRemoved func(peer.ID)
	PeerAdded   func(peer.ID)

	// usefulnessGracePeriod is the maximum grace period we will give to a
	// peer in the bucket to be useful to us, failing which, we will evict
	// it to make place for a new peer if the bucket is full
	usefulnessGracePeriod time.Duration

	df *peerdiversity.Filter

	//Added by Kanemitsu START
	/**
	STORE message arrival rate
	derived by : num_arrive / rtt_duration
	*/
	arv_rate_store float64

	/**
	# of STORE(addPeer) requests
	 */
	num_arrive int64


	/**
	Pool Size
	*/
	pool_size int

	/**
	k-bucket entry exchange probability
	that is derived by :
	num_exchange / num_arrive per rttInterval
	*/
	prob_exchange float64

	num_exchange int64
	//Added by Kanemitsu END

	isKadRTT bool

	/**
	Time interval to obtain RTT
	 */
	rttInterval time.Duration

	lastExTime time.Time

}

// NewRoutingTable creates a new routing table with a given bucketsize, local ID, and latency tolerance.
func NewRoutingTable(bucketsize int, localID ID, latency time.Duration, m peerstore.Metrics, usefulnessGracePeriod time.Duration,
	df *peerdiversity.Filter) (*RoutingTable, error) {
	rt := &RoutingTable{
		buckets:    []*bucket{newBucket()},
		bucketsize: bucketsize,
		local:      localID,

		maxLatency: latency,
		metrics:    m,

		cplRefreshedAt: make(map[uint]time.Time),

		PeerRemoved: func(peer.ID) {},
		PeerAdded:   func(peer.ID) {},

		usefulnessGracePeriod: usefulnessGracePeriod,

		df: df,

	}
	rt.isKadRTT = true
	//Addec by Kanemitsu START
	rt.arv_rate_store = 0.01
	rt.pool_size = rt.bucketsize
	rt.prob_exchange = 1

	rt.setOptValues(0)
	//set the initial values for k, alpha, and beta.
	//rt.buildInitParameters()
	rt.lastExTime = time.Now()
	rt.num_arrive = 0
	rt.num_exchange = 0

	//Addec by Kanemitsu END

	rt.ctx, rt.ctxCancel = context.WithCancel(context.Background())

/*
	a := []byte{0,0}
	b := []byte{1,0}
	cpl := CommonPrefixLen(a,b)
	//cpl2 := ks.ZeroPrefixLen(u.XOR(a, b))
	cpl2 := u.XOR(a, b)

	fmt.Printf("XOR:%d / Len:%d", cpl2, cpl)
*/
	return rt, nil
}

func (rt *RoutingTable) SetRTT(p peer.ID, rtt time.Duration){
	//pに対して，rttをセットする．
	bucketID := rt.bucketIdForPeer(p)
	//bucket := rt.buckets[bucketID]
	bucket := rt.GetBucket(bucketID)
	peer := bucket.getPeer(p)
	peer.SetRTT(rtt)
}

func (rt *RoutingTable) GetRTT(p peer.ID) time.Duration {
	bucketID := rt.bucketIdForPeer(p)
	//bucket := rt.buckets[bucketID]
	bucket := rt.GetBucket(bucketID)
	peer := bucket.getPeer(p)

	return peer.GetRTT()
}



func (rt *RoutingTable) configPool() {
	maxPoolSize:= 0

	for i := 0; i< len(rt.buckets); i++{
		br := rt.buckets[i]
		if(maxPoolSize <= br.k){
			maxPoolSize = br.k
		}
	}
	//rt.setPoolSize(maxPoolSize)
	rt.setPoolSize(rt.bucketsize)
}


func (rt *RoutingTable) setOptValues(idx int) *bucket {
	initB := rt.buckets[idx]
	k_opt := rt.CalcKOpt(idx)

	k_opt = int(math.Max(float64(k_opt), float64(rt.bucketsize)))

	initB.SetK(k_opt)



	b_opt := rt.CalcBetaOpt(idx)
	if(b_opt < 1){
		b_opt = k_opt
	}
	b_opt = int(math.Min(float64(k_opt), float64(b_opt) ))

	initB.SetBeta(b_opt)



	a_opt := rt.CalcAlphaOpt(idx)
	if( a_opt > int(rt.pool_size)){
		a_opt = int(rt.pool_size)
	}
	a_opt = int(math.Max(2, float64(a_opt)))
	//a_opt = int(math.Max(rt.alpha), )

	initB.SetAlpha(a_opt)
	//fmt.Println("***OptA: %d", a_opt)
	rt.configPool()

	return initB
}

func (rt*RoutingTable) GetBuckets() []*bucket {
	return rt.buckets
}



func (rt*RoutingTable) GetBucket(cpl int) *bucket {
	return rt.buckets[cpl]
}

func (rt *RoutingTable) setStoreRate(val float64){
	rt.arv_rate_store = val
}

func (rt*RoutingTable) setProbExchange(val float64){
	rt.prob_exchange = val
}

func (rt*RoutingTable) setPoolSize(val int){
	rt.pool_size = val
}

//Derive optimal k
func (rt *RoutingTable) CalcKOpt(idx int) int{
	r := idx + 1
	val := (float64)(rt.prob_exchange * rt.arv_rate_store)
	w2  := math.Log(val)
	w1 := -1 * math.Pow(2, float64(r)) * math.Pow(math.E, (float64)(-1 * rt.prob_exchange*rt.arv_rate_store))
	nomi := rt.LambertW(0, w1*w2)
	denomi := math.Log((float64)(rt.prob_exchange * rt.arv_rate_store))
	k_opt := int(math.Max(float64(1),math.Floor(-1*nomi/denomi)))
	if k_opt < 2 {
		k_opt = 2
	}


	return k_opt

}

func (rt *RoutingTable)LambertW(k int, x float64) float64 {
	// Special cases.
	switch {
	case k < -1 || k > 0 || x < -1/math.E || (k == -1 && x > 0) || math.IsNaN(x):
		return math.NaN()
	case x == 0:
		if k == 0 {
			return 0
		}
		return math.Inf(-1)
	case x == -1/math.E:
		return -1
	case math.IsInf(x, 1):
		return x
	}

	// Estimate an initial value using approximations and then use
	// Fritsch iteration (once) to get an improved estimate with O(1e-15) error
	w := rt.initial(k, x)
	return rt.fritsch(w, x)
}

func (rt *RoutingTable)fritsch(w, x float64) float64 {
	z := math.Log(x/w) - w
	w1 := w + 1
	q := 2 * w1 * (w1 + 2*z/3)
	eps := z / w1 * (q - z) / (q - 2*z)
	return w * (1 + eps)
}

func (rt *RoutingTable)initial(k int, x float64) float64 {
	switch k {
	case 0:
		const (
			xbranch = -0.32358170806015724
			xratp0  = 0.14546954290661823
			xratp1  = 8.706658967856612
		)
		switch {
		case x < xbranch:
			return rt.branchpoint(k, x)
		case x < xratp0:
			return rt.rationalp0(x)
		case x < xratp1:
			return rt.rationalp1(x)
		default:
			return rt.asymptotic(k, x)
		}
	default: // k=-1
		const xbranch = -0.30298541769
		switch {
		case x < xbranch:
			return rt.branchpoint(k, x)
		default:
			return rt.rationalm(x)
		}
	}
}

func (rt *RoutingTable)rationalm(x float64) float64 {
	const (
		a0 = -7.81417672390744
		a1 = 253.88810188892484
		a2 = 657.9493176902304

		b0 = 1
		b1 = -60.43958713690808
		b2 = 99.9856708310761
		b3 = 682.6073999909428
		b4 = 962.1784396969866
		b5 = 1477.9341280760887
	)

	return (a0 + x*(a1+x*a2)) / (b0 + x*(b1+x*(b2+x*(b3+x*(b4+x*b5)))))
}
// asymptotic returns an asymptotic estimate of W(x, k)
func (rt *RoutingTable)asymptotic(k int, x float64) float64 {
	s := 1 + 2*k
	a := math.Log(float64(s) * x)
	b := math.Log(float64(s) * a)

	ba := b / a
	b2 := b * b
	b3 := b2 * b
	b4 := b2 * b2

	q0 := b - 2
	q1 := 2*b2 - 9*b + 6
	q2 := 3*b3 - 22*b2 + 36*b - 12
	q3 := 12*b4 - 125*b3 + 350*b2 - 300*b + 60
	return a - b + ba*(1+1/(2*a)*(q0+1/(3*a)*(q1+1/(2*a)*(q2+1/(5*a)*q3))))
}
func (rt *RoutingTable) rationalp0(x float64) float64 {
	const (
		a0 = 1
		a1 = 5.931375839364438
		a2 = 11.39220550532913
		a3 = 7.33888339911111
		a4 = 0.653449016991959

		b0 = 1
		b1 = 6.931373689597704
		b2 = 16.82349461388016
		b3 = 16.43072324143226
		b4 = 5.115235195211697
	)
	num := a0 + x*(a1+x*(a2+x*(a3+x*a4)))
	den := b0 + x*(b1+x*(b2+x*(b3+x*b4)))
	return x * num / den
}

func (rt *RoutingTable) rationalp1(x float64) float64 {
	const (
		a0 = 1
		a1 = 2.445053070726557
		a2 = 1.343664225958226
		a3 = 0.148440055397592
		a4 = 0.0008047501729130

		b0 = 1
		b1 = 3.444708986486002
		b2 = 3.292489857371952
		b3 = 0.916460018803122
		b4 = 0.0530686404483322
	)
	num := a0 + x*(a1+x*(a2+x*(a3+x*a4)))
	den := b0 + x*(b1+x*(b2+x*(b3+x*b4)))
	return x * num / den
}
func (rt *RoutingTable)branchpoint(k int, x float64) float64 {
	s := 1 + 2*k
	p := float64(s) * math.Sqrt2 * math.Sqrt(1+math.E*x)
	const (
		b0 = -1
		b1 = 1
		b2 = -0.3333333333333333
		b3 = 0.1527777777777778
		b4 = -0.07962962962962963
		b5 = 0.04450231481481481
		b6 = -0.02598471487360376
		b7 = 0.01563563253233392
		b8 = -0.009616892024299432
		b9 = 0.006014543252956118
	)
	return b0 + p*(b1+p*(b2+p*(b3+p*(b4+p*(b5+p*(b6+p*(b7+p*(b8+p*b9))))))))
}


//Derive optimal alpha
func (rt *RoutingTable) CalcAlphaOpt(idx int) int{

	//	b0 := rt.buckets[0]
	//k0 := b0.k
	br := rt.buckets[idx]

	var p_query float64
	var pro float64
	pro = 1.0
	b_pre := rt.buckets[0]
	if idx > 0 {
		b_pre = rt.buckets[idx - 1]
	}else{
		b_pre = rt.buckets[0]
	}

	pre_pro := b_pre.p_not

	pro = pre_pro *  float64( float64(1)- float64(1/float64(b_pre.k)))
	br.p_not = pro
	if(idx == 0){
		pre_pro = 1
		br.p_not = 1
	}
	//p_query = b_pre.p_query + pro * 1/float64(br.k)
	p_query =  pro * 1/float64(br.k)
	br.p_query = p_query

	//p_query = math.Min(0.99, p_query)
	p_not := float64(1.0 - p_query)

	br.p_not = pro * float64( float64(1)- float64(1/float64(br.k)))

	denomi := float64(float64(br.beta) * float64(br.k) * float64(rt.pool_size) * math.Log(p_not))

	w := rt.LambertW(0, float64(-1)*(float64(rt.pool_size)*math.Pow(p_not, float64(rt.pool_size) * float64(br.beta)+ (float64(rt.pool_size)*float64(br.beta))/(1- math.Pow(p_not, float64(rt.pool_size)*float64(br.beta))))*float64(br.beta) *math.Log(p_not))/(math.Pow(p_not, float64(rt.pool_size * br.beta) )-1))
	w2 := (float64(br.beta) * float64(rt.pool_size)* math.Log(p_not))/(1- math.Pow(p_not, float64(br.beta) * float64(rt.pool_size)))
	alpha_opt := math.Abs(float64(math.Ceil(denomi / float64((w - w2)))))

	if (alpha_opt < 2){
		alpha_opt = 2
	}
	return int(math.Ceil(alpha_opt))
}

//Derive optimal beta
func (rt *RoutingTable) CalcBetaOpt(idx int) int{
	/*br := rt.buckets[idx]
	alphaR := br.alpha
	beta_opt := rt.pool_size/alphaR
	*/
	br := rt.buckets[idx]

	//b0 := rt.buckets[0]
	//beta_opt := int(rt.pool_size * b0.k / br.alpha)
	beta_opt := int(math.Min(float64(br.k), float64(rt.pool_size)))


	return beta_opt
}


//set the initial alpha, beta, and k for each k-bucket in KadRTT
func (rt *RoutingTable) buildInitParameters(){

	for i:=0 ; i < 160; i++{
		rt.nextBucket()
		idx := len(rt.buckets)-1
	//	rt.setOptValues(idx)
		b := rt.buckets[idx]

		fmt.Printf("idx:%d / k: %d, alpha:%d, beta: %d , pool: %d\n", i, b.k, b.alpha, b.beta, rt.pool_size)
		//fmt.Print("k_opt:", k)
	}
}

// Close shuts down the Routing Table & all associated processes.
// It is safe to call this multiple times.
func (rt *RoutingTable) Close() error {
	rt.ctxCancel()
	return nil
}

// NPeersForCPL returns the number of peers we have for a given Cpl
func (rt *RoutingTable) NPeersForCpl(cpl uint) int {
	rt.tabLock.RLock()
	defer rt.tabLock.RUnlock()

	// it's in the last bucket
	if int(cpl) >= len(rt.buckets)-1 {
		count := 0
		b := rt.buckets[len(rt.buckets)-1]
		for _, p := range b.peers() {
			if CommonPrefixLen(rt.local, p.dhtId) == int(cpl) {
				count++
			}
		}
		return count
	} else {
		return rt.buckets[cpl].len()
	}
}


// TryAddPeer tries to add a peer to the Routing table.
// If the peer ALREADY exists in the Routing Table and has been queried before, this call is a no-op.
// If the peer ALREADY exists in the Routing Table but hasn't been queried before, we set it's LastUsefulAt value to
// the current time. This needs to done because we don't mark peers as "Useful"(by setting the LastUsefulAt value)
// when we first connect to them.
//
// If the peer is a queryPeer i.e. we queried it or it queried us, we set the LastSuccessfulOutboundQuery to the current time.
// If the peer is just a peer that we connect to/it connected to us without any DHT query, we consider it as having
// no LastSuccessfulOutboundQuery.
//
//
// If the logical bucket to which the peer belongs is full and it's not the last bucket, we try to replace an existing peer
// whose LastSuccessfulOutboundQuery is above the maximum allowed threshold in that bucket with the new peer.
// If no such peer exists in that bucket, we do NOT add the peer to the Routing Table and return error "ErrPeerRejectedNoCapacity".

// It returns a boolean value set to true if the peer was newly added to the Routing Table, false otherwise.
// It also returns any error that occurred while adding the peer to the Routing Table. If the error is not nil,
// the boolean value will ALWAYS be false i.e. the peer wont be added to the Routing Table it it's not already there.
//
// A return value of false with error=nil indicates that the peer ALREADY exists in the Routing Table.
func (rt *RoutingTable) TryAddPeer(p peer.ID, queryPeer bool, isReplaceable bool) (bool, error) {
	rt.tabLock.Lock()
	defer rt.tabLock.Unlock()

	return rt.addPeer(p, queryPeer, isReplaceable)
}

func (rt *RoutingTable) TryAddPeerKadRTT(p peer.ID, queryPeer bool, isReplaceable bool, rtt time.Duration) (bool, error) {
	rt.tabLock.Lock()
	defer rt.tabLock.Unlock()

		return rt.addPeerKadRTT(p, queryPeer, isReplaceable, rtt)



}

func Distance(k1, k2 []byte) *big.Int {
	// XOR the keys
	k3 := u.XOR(k1, k2)

	// interpret it as an integer
	dist := big.NewInt(0).SetBytes(k3)
	return dist
}


//Added by Kanemitsu
func (rt *RoutingTable) calcIDVariance(b *bucket, p peer.ID, rtt time.Duration) *big.Int{
	//Sort by increasing order of key
	var bp ByPeer = b.peers()
	//peers := ks.ByPeer(bp)
	len := len(b.peers())
	//tmpb := b
	sort.Sort(bp)
	//var total *big.Int
	var total = big.NewInt(0)
	rttList := list.New()

	for i:=0;i<len;i++ {
		//If the exising entry's RTT > new Entry's RTT,
		//the existing one is put into the list
		if(bp[i].rtt >= rtt ){
			rttList.PushFront(bp[i])
		}
		if(i == len-1){
			break;
		}
		//Calc ID Distance for two entries
		dist := Distance(ConvertPeerID(bp[i].Id), ConvertPeerID(bp[i+1].Id))
		total.Add(total, dist)

	}


	var vTotal = big.NewInt(0)
	//var variance *big.Int
	avg := total.Div(total, big.NewInt(int64(len)))
	for i:=0;i<len-1;i++ {
		//Calc ID Distance for two entries
		//dist := Distance(ConvertPeerID(bp[i].Id),ConvertPeerID(bp[i+1].Id))
		//total.Add(total, dist)
		dist := Distance(ConvertPeerID(bp[i].Id), ConvertPeerID(bp[i+1].Id))
		sub := total.Abs(total.Sub(dist, avg))
		m := sub.Mul(sub, sub)
		vTotal.Add(vTotal, m)
	}


	retP := p
	retP.Size()


	//_ = rttList.Front().Value.(*PeerInfo).Id
	tmpB := newBucket()
	copy(tmpB.peers(), b.peers())

	for e := rttList.Front(); e!= nil; e = e.Next() {
		oldP := e.Value.(PeerInfo).Id
		//Calc ID Variance for each peer RTT > new Peer RTT
		newV := rt.getIDVariance( b, oldP, p, rtt)
		if(newV.Cmp(vTotal) < 0){
			vTotal = newV
			retP = oldP
		}
	}

	//If the actual ID is set to retP,
	//the bucket should be updated.
	if(retP != p ) {
		b.remove(retP)
		//Then, the new one is added.
		b.pushFront(&PeerInfo{
			Id:                            p,
			LastUsefulAt:                  time.Now(),
			LastSuccessfulOutboundQueryAt: time.Now(),
			AddedAt:                       time.Now(),
			dhtId:                         ConvertPeerID(p),
			replaceable:                   true,
			rtt:                           rtt,
		})

		tmpB.pushFront(&PeerInfo{
			Id:                            p,
			LastUsefulAt:                  time.Now(),
			LastSuccessfulOutboundQueryAt: time.Now(),
			AddedAt:                       time.Now(),
			dhtId:                         ConvertPeerID(p),
			replaceable:                   true,
			rtt:                           rtt,
		})
		rt.prob_exchange++
		rt.PeerAdded(p)


	}else{

	}
	b.idVariance = vTotal

	return vTotal

}

func remove(slice []PeerInfo, s int) []PeerInfo {
	return append(slice[:s], slice[s+1:]...)
}


//Added by Kanemitsu
func (rt *RoutingTable) getIDVariance(b *bucket, oldP peer.ID, newP peer.ID, rtt time.Duration) *big.Int {
	var total = big.NewInt(0)
	//tmpB := b
	tmpB := newBucket()
	copy(tmpB.peers(), b.peers())

	tmpB.remove(oldP)
	tmpB.pushFront(&PeerInfo{
		Id:                            newP,
		LastUsefulAt:                  time.Now(),
		LastSuccessfulOutboundQueryAt: time.Now(),
		AddedAt:                       time.Now(),
		dhtId:                         ConvertPeerID(newP),
		replaceable:                   true,
		rtt:                           rtt,
	})

	len := len(tmpB.peers())

	var bp ByPeer = tmpB.peers()
	//peers := ks.ByPeer(bp)

	sort.Sort(bp)
	//var total *big.Int


	for i:=0;i<len;i++ {
		//If the exising entry's RTT > new Entry's RTT,
		//the existing one is put into the list
		if(i == len-1){
			break;
		}
		//Calc ID Distance for two entries
		dist := Distance(ConvertPeerID(bp[i].Id), ConvertPeerID(bp[i+1].Id))

		total.Add(total, dist)

	}

	var vTotal = big.NewInt(0)
	//var variance *big.Int
	avg := total.Div(total, big.NewInt(int64(len)))
	for i:=0;i<len-1;i++ {
		//Calc ID Distance for two entries
		//dist := Distance(ConvertPeerID(bp[i].Id),ConvertPeerID(bp[i+1].Id))
		//total.Add(total, dist)
		dist := Distance(ConvertPeerID(bp[i].Id), ConvertPeerID(bp[i+1].Id))
		sub := total.Abs(total.Sub(dist, avg))
		m := sub.Mul(sub, sub)
		vTotal.Add(vTotal, m)
	}


	return vTotal
}


//Added by Kanemitsu
func (rt *RoutingTable) getIDVarianceWithOut(b ByPeer, id peer.ID) *big.Int {
	var total = big.NewInt(0)
	//tmpB := b
	tmpB := newBucket()
	copy(tmpB.peers(), b)
	tmpB.remove(id)

	len := len(tmpB.peers())

	var bp ByPeer = tmpB.peers()
	//peers := ks.ByPeer(bp)

	sort.Sort(bp)
	//var total *big.Int


	for i:=0;i<len;i++ {
		//If the exising entry's RTT > new Entry's RTT,
		//the existing one is put into the list
		if(i == len-1){
			break;
		}
		//Calc ID Distance for two entries
		dist := Distance(ConvertPeerID(bp[i].Id), ConvertPeerID(bp[i+1].Id))
		total.Add(total, dist)

	}

	var vTotal = big.NewInt(0)
	//var variance *big.Int
	avg := total.Div(total, big.NewInt(int64(len)))
	for i:=0;i<len-1;i++ {
		//Calc ID Distance for two entries
		//dist := Distance(ConvertPeerID(bp[i].Id),ConvertPeerID(bp[i+1].Id))
		//total.Add(total, dist)
		dist := Distance(ConvertPeerID(bp[i].Id), ConvertPeerID(bp[i+1].Id))
		sub := total.Abs(total.Sub(dist, avg))
		m := sub.Mul(sub, sub)
		vTotal.Add(vTotal, m)
	}


	return vTotal
}

func (rt *RoutingTable) addPeerKadRTT(p peer.ID, queryPeer bool, isReplaceable bool, rtt time.Duration) (bool, error) {
	bucketID := rt.bucketIdForPeer(p)
	bucket := rt.buckets[bucketID]
	rtt = rt.metrics.LatencyEWMA(p)


	now := time.Now()
	var lastUsefulAt time.Time
	if queryPeer {
		lastUsefulAt = now
	}

	//Update the exchange probability
	//At first, increment the # of arrivals.
	rt.num_arrive++
	span := time.Since(rt.lastExTime)
	if span.Seconds() >= rt.rttInterval.Seconds() {

		//old_alpha := bucket.alpha
		//old_beta := bucket.beta
		//old_k := bucket.k
		rt.arv_rate_store = float64(float64(rt.num_arrive) / float64(span))
		rt.prob_exchange = float64(float64(rt.num_exchange) / float64(rt.num_arrive))

		//Update optimal values for alpha, beta, k for the specific k-bucket index.
		bucket = rt.setOptValues(bucketID)

		//bucket = rt.GetBucket(bucketID)
		if(bucket.len() < bucket.k){
			rt.prob_exchange = 1
		}


		if( bucket.len() > bucket.k){
			//if the number of bucket entries is reduced, we must
			//select members to be discarded.
			//select the by which the ID variance is large.
			num := bucket.len() - bucket.k

			var bp ByPeer = bucket.peers()
			//peers := ks.ByPeer(bp)
			peer_num := len(bucket.peers())

			//tmpb := b
			//IDの小さい順にソートする．
			sort.Sort(bp)
			retList := list.New()

			for i:=0;i<peer_num;i++ {
				//bp[i]をとった場合のID分散値を求める．
				//オブジェクトを定義して，それにID分散値をもたせる．
				//そのオブジェクトのリストを作り，さらにID分散値
				//の降順にソートする．
				//そのあと，numの数だけ取り出して，bucketからremoveする．
				variance := rt.getIDVarianceWithOut(bp, bp[i].Id)
				retList.PushFront(&VarianceInfo{
					id:  	bp[i].Id,
					variance:	variance,
				})
			}
			//fmt.Println("#### Before:%d", len(bucket.peers()))
			//Remove $num peers from the k-bucket.
			for i:=0;i<num;i++ {
				targetID := retList.Front()
				bucket.remove(targetID.Value.(VarianceInfo).id)
				retList.Remove(targetID)
			}
			//fmt.Println("#### After:%d", len(bucket.peers()))


		}else{
			//If the number is expanded, no need for reducing the entries in
			// the bucket.
		}
		//Reset the values of exchange probability for the next calculation
		rt.lastExTime = time.Now()
		rt.num_arrive = 0
		rt.num_exchange = 0


	}


	// peer already exists in the Routing Table.
	if peer := bucket.getPeer(p); peer != nil {
		// if we're querying the peer first time after adding it, let's give it a
		// usefulness bump. This will ONLY happen once.
		if peer.LastUsefulAt.IsZero() && queryPeer {
			peer.LastUsefulAt = lastUsefulAt
		}
		return false, nil
	}

	//fmt.Print("****RTT:", rtt)

	//fmt.Println("***currentLatency: %d / Max:%d: ", rt.metrics.LatencyEWMA(p), rt.maxLatency)
	// peer's latency threshold is NOT acceptable
	if rt.metrics.LatencyEWMA(p) > rt.maxLatency {
		// Connection doesnt meet requirements, skip!
		return false, ErrPeerRejectedHighLatency
	}
	//以降は，ローカルにpが無い状況．

	// add it to the diversity filter for now.
	// if we aren't able to find a place for the peer in the table,
	// we will simply remove it from the Filter later.
	if rt.df != nil {
		if !rt.df.TryAdd(p) {
			return false, errors.New("peer rejected by the diversity filter")
		}
	}
	bsize := rt.bucketsize
	if(rt.isKadRTT){
		bsize = bucket.k
	}


	// We have enough space in the bucket (whether spawned or grouped).
	if bucket.len() < bsize {
		bucket.pushFront(&PeerInfo{
			Id:                            p,
			LastUsefulAt:                  lastUsefulAt,
			LastSuccessfulOutboundQueryAt: now,
			AddedAt:                       now,
			dhtId:                         ConvertPeerID(p),
			replaceable:                   isReplaceable,
			rtt:                           rtt,
		})
		rt.PeerAdded(p)


		return true, nil
	}


	//in case of k-bucket is full
	if bucketID == len(rt.buckets)-1 {
		// if the bucket is too large and this is the last bucket (i.e. wildcard), unfold it.
		rt.nextBucket()
		// the structure of the table has changed, so let's recheck if the peer now has a dedicated bucket.
		bucketID = rt.bucketIdForPeer(p)
		bucket = rt.buckets[bucketID]
		blen := rt.bucketsize
		if(rt.isKadRTT){
			blen = bucket.k
		}
		// push the peer only if the bucket isn't overflowing after slitting
		if bucket.len() < blen/*rt.bucketsize*/ {
			bucket.pushFront(&PeerInfo{
				Id:                            p,
				LastUsefulAt:                  lastUsefulAt,
				LastSuccessfulOutboundQueryAt: now,
				AddedAt:                       now,
				dhtId:                         ConvertPeerID(p),
				replaceable:                   isReplaceable,
				rtt:                           rtt,
			})
			rt.PeerAdded(p)
			return true, nil
		}
	}

	//the case that k-bucket is full.
	//Added by Kanemitsu
	if rt.isKadRTT {

		//If the k-bucket is full, ID-rearrengement is performed.
		//If the ID variance is made lower, the exchange is accepted.
		//To do that, the current ID variance is calculated when addPeer is called.
		//IDVariance := rt.calcIDVariance(bucket, bucketID, p, rtt)
		if ( bucket.len() == 1){
			info := bucket.list.Front().Value.(*PeerInfo)
			//fmt.Println("*****firstRTT: %d, NextRTT, %d", info.rtt, rtt)

			if (info.rtt >= rtt){
				//swap
				rt.removePeer(info.Id)
				bucket.pushFront(&PeerInfo{
					Id:                            p,
					LastUsefulAt:                  lastUsefulAt,
					LastSuccessfulOutboundQueryAt: now,
					AddedAt:                       now,
					dhtId:                         ConvertPeerID(p),
					replaceable:                   isReplaceable,
					rtt:                           rtt,
				})
				rt.PeerAdded(p)
				rt.num_exchange++
				//return true, nil
			}

		}else{
			rt.calcIDVariance(bucket, p, rtt)
		}
		//Added by Kanemitsu
	/*	replaceablePeer := bucket.min(func(p1 *PeerInfo, p2 *PeerInfo) bool {
			return p1.replaceable
		})

		if replaceablePeer != nil && replaceablePeer.replaceable {
			// let's evict it and add the new peer
			if rt.removePeer(replaceablePeer.Id) {
				bucket.pushFront(&PeerInfo{
					Id:                            p,
					LastUsefulAt:                  lastUsefulAt,
					LastSuccessfulOutboundQueryAt: now,
					AddedAt:                       now,
					dhtId:                         ConvertPeerID(p),
					replaceable:                   isReplaceable,
				})
				rt.PeerAdded(p)
				return true, nil
			}
		}
*/
	}else{
		// the bucket to which the peer belongs is full. Let's try to find a peer
		// in that bucket which is replaceable.
		// we don't really need a stable sort here as it dosen't matter which peer we evict
		// as long as it's a replaceable peer.
		replaceablePeer := bucket.min(func(p1 *PeerInfo, p2 *PeerInfo) bool {
			return p1.replaceable
		})

		if replaceablePeer != nil && replaceablePeer.replaceable {
			// let's evict it and add the new peer
			if rt.removePeer(replaceablePeer.Id) {
				bucket.pushFront(&PeerInfo{
					Id:                            p,
					LastUsefulAt:                  lastUsefulAt,
					LastSuccessfulOutboundQueryAt: now,
					AddedAt:                       now,
					dhtId:                         ConvertPeerID(p),
					replaceable:                   isReplaceable,
				})
				rt.PeerAdded(p)
				return true, nil
			}
		}
	}


	// we weren't able to find place for the peer, remove it from the filter state.
	if rt.df != nil {
		rt.df.Remove(p)
	}
	return false, ErrPeerRejectedNoCapacity
}
// locking is the responsibility of the caller
func (rt *RoutingTable) addPeer(p peer.ID, queryPeer bool, isReplaceable bool) (bool, error) {
	bucketID := rt.bucketIdForPeer(p)
	bucket := rt.buckets[bucketID]

	now := time.Now()
	var lastUsefulAt time.Time
	if queryPeer {
		lastUsefulAt = now
	}


	// peer already exists in the Routing Table.
	if peer := bucket.getPeer(p); peer != nil {
		// if we're querying the peer first time after adding it, let's give it a
		// usefulness bump. This will ONLY happen once.
		if peer.LastUsefulAt.IsZero() && queryPeer {
			peer.LastUsefulAt = lastUsefulAt
		}
		return false, nil
	}
	//fmt.Println("****Latency: %d / Max: %d", rt.metrics.LatencyEWMA(p),rt.maxLatency )
	// peer's latency threshold is NOT acceptable
	if rt.metrics.LatencyEWMA(p) > rt.maxLatency {
		// Connection doesnt meet requirements, skip!
		return false, ErrPeerRejectedHighLatency
	}
//以降は，ローカルにpが無い状況．

	// add it to the diversity filter for now.
	// if we aren't able to find a place for the peer in the table,
	// we will simply remove it from the Filter later.
	if rt.df != nil {
		if !rt.df.TryAdd(p) {
			return false, errors.New("peer rejected by the diversity filter")
		}
	}
	bsize := rt.bucketsize
	if(rt.isKadRTT){
		bsize = bucket.k
	}


	//まずはpingを行って遅延を計測する．


	// We have enough space in the bucket (whether spawned or grouped).
	if bucket.len() < bsize {
			bucket.pushFront(&PeerInfo{
				Id:                            p,
				LastUsefulAt:                  lastUsefulAt,
				LastSuccessfulOutboundQueryAt: now,
				AddedAt:                       now,
				dhtId:                         ConvertPeerID(p),
				replaceable:                   isReplaceable,
			})
			rt.PeerAdded(p)


		return true, nil
	}

	//Bucketがフルの場合
	if bucketID == len(rt.buckets)-1 {
		// if the bucket is too large and this is the last bucket (i.e. wildcard), unfold it.
		rt.nextBucket()
		// the structure of the table has changed, so let's recheck if the peer now has a dedicated bucket.
		bucketID = rt.bucketIdForPeer(p)
		bucket = rt.buckets[bucketID]
		blen := rt.bucketsize
		if(rt.isKadRTT){
			blen = bucket.k
		}
		// push the peer only if the bucket isn't overflowing after slitting
		if bucket.len() < blen/*rt.bucketsize*/ {
			bucket.pushFront(&PeerInfo{
				Id:                            p,
				LastUsefulAt:                  lastUsefulAt,
				LastSuccessfulOutboundQueryAt: now,
				AddedAt:                       now,
				dhtId:                         ConvertPeerID(p),
				replaceable:                   isReplaceable,
			})
			rt.PeerAdded(p)
			return true, nil
		}
	}

	if rt.isKadRTT {
		//以降は，フルのとき．
		//もし新規ピアpを入れることで，ID距離の分散が小さくなるのであれば交換する．
		//そのために，現状の分散値を取得する．


	}else{
		// the bucket to which the peer belongs is full. Let's try to find a peer
		// in that bucket which is replaceable.
		// we don't really need a stable sort here as it dosen't matter which peer we evict
		// as long as it's a replaceable peer.
		replaceablePeer := bucket.min(func(p1 *PeerInfo, p2 *PeerInfo) bool {
			return p1.replaceable
		})

		if replaceablePeer != nil && replaceablePeer.replaceable {
			// let's evict it and add the new peer
			if rt.removePeer(replaceablePeer.Id) {
				bucket.pushFront(&PeerInfo{
					Id:                            p,
					LastUsefulAt:                  lastUsefulAt,
					LastSuccessfulOutboundQueryAt: now,
					AddedAt:                       now,
					dhtId:                         ConvertPeerID(p),
					replaceable:                   isReplaceable,
				})
				rt.PeerAdded(p)
				return true, nil
			}
		}
	}


	// we weren't able to find place for the peer, remove it from the filter state.
	if rt.df != nil {
		rt.df.Remove(p)
	}
	return false, ErrPeerRejectedNoCapacity
}

// MarkAllPeersIrreplaceable marks all peers in the routing table as irreplaceable
// This means that we will never replace an existing peer in the table to make space for a new peer.
// However, they can still be removed by calling the `RemovePeer` API.
func (rt *RoutingTable) MarkAllPeersIrreplaceable() {
	rt.tabLock.Lock()
	defer rt.tabLock.Unlock()

	for i := range rt.buckets {
		b := rt.buckets[i]
		b.updateAllWith(func(p *PeerInfo) {
			p.replaceable = false
		})
	}
}

// GetPeerInfos returns the peer information that we've stored in the buckets
func (rt *RoutingTable) GetPeerInfos() []PeerInfo {
	rt.tabLock.RLock()
	defer rt.tabLock.RUnlock()

	var pis []PeerInfo
	for _, b := range rt.buckets {
		for _, p := range b.peers() {
			pis = append(pis, p)
		}
	}
	return pis
}

// UpdateLastSuccessfulOutboundQuery updates the LastSuccessfulOutboundQueryAt time of the peer.
// Returns true if the update was successful, false otherwise.
func (rt *RoutingTable) UpdateLastSuccessfulOutboundQueryAt(p peer.ID, t time.Time) bool {
	rt.tabLock.Lock()
	defer rt.tabLock.Unlock()

	bucketID := rt.bucketIdForPeer(p)
	bucket := rt.buckets[bucketID]

	if pc := bucket.getPeer(p); pc != nil {
		pc.LastSuccessfulOutboundQueryAt = t
		return true
	}
	return false
}

// UpdateLastUsefulAt updates the LastUsefulAt time of the peer.
// Returns true if the update was successful, false otherwise.
func (rt *RoutingTable) UpdateLastUsefulAt(p peer.ID, t time.Time) bool {
	rt.tabLock.Lock()
	defer rt.tabLock.Unlock()

	bucketID := rt.bucketIdForPeer(p)
	bucket := rt.buckets[bucketID]

	if pc := bucket.getPeer(p); pc != nil {
		pc.LastUsefulAt = t
		return true
	}
	return false
}

// RemovePeer should be called when the caller is sure that a peer is not useful for queries.
// For eg: the peer could have stopped supporting the DHT protocol.
// It evicts the peer from the Routing Table.
func (rt *RoutingTable) RemovePeer(p peer.ID) {
	rt.tabLock.Lock()
	defer rt.tabLock.Unlock()
	rt.removePeer(p)
}

// locking is the responsibility of the caller
func (rt *RoutingTable) removePeer(p peer.ID) bool {
	bucketID := rt.bucketIdForPeer(p)
	bucket := rt.buckets[bucketID]
	if bucket.remove(p) {
		if rt.df != nil {
			rt.df.Remove(p)
		}
		for {
			lastBucketIndex := len(rt.buckets) - 1

			// remove the last bucket if it's empty and it isn't the only bucket we have
			if len(rt.buckets) > 1 && rt.buckets[lastBucketIndex].len() == 0 {
				rt.buckets[lastBucketIndex] = nil
				rt.buckets = rt.buckets[:lastBucketIndex]
			} else if len(rt.buckets) >= 2 && rt.buckets[lastBucketIndex-1].len() == 0 {
				// if the second last bucket just became empty, remove and replace it with the last bucket.
				rt.buckets[lastBucketIndex-1] = rt.buckets[lastBucketIndex]
				rt.buckets[lastBucketIndex] = nil
				rt.buckets = rt.buckets[:lastBucketIndex]
			} else {
				break
			}
		}

		// peer removed callback
		rt.PeerRemoved(p)
		return true
	}
	return false
}

func (rt *RoutingTable) nextBucket() {
	// This is the last bucket, which allegedly is a mixed bag containing peers not belonging in dedicated (unfolded) buckets.
	// _allegedly_ is used here to denote that *all* peers in the last bucket might feasibly belong to another bucket.
	// This could happen if e.g. we've unfolded 4 buckets, and all peers in folded bucket 5 really belong in bucket 8.
	bucket := rt.buckets[len(rt.buckets)-1]
	newBucket := bucket.split(len(rt.buckets)-1, rt.local)

	rt.buckets = append(rt.buckets, newBucket)

	//Added by Kanemitsu
	//newBucket = rt.setOptValues(len(rt.buckets)-1)
	bsize := rt.bucketsize
	if rt.isKadRTT {
		bsize = newBucket.k
		idx := len(rt.buckets)-1
		rt.setOptValues(idx)
		//b := rt.buckets[idx]
	}
	// The newly formed bucket still contains too many peers. We probably just unfolded a empty bucket.
	if newBucket.len() >= bsize {
		// Keep unfolding the table until the last bucket is not overflowing.
		rt.nextBucket()
	}
}

// Find a specific peer by ID or return nil
func (rt *RoutingTable) Find(id peer.ID) peer.ID {
	srch := rt.NearestPeers(ConvertPeerID(id), 1)
	if len(srch) == 0 || srch[0] != id {
		return ""
	}
	return srch[0]
}

// NearestPeer returns a single peer that is nearest to the given ID
func (rt *RoutingTable) NearestPeer(id ID) peer.ID {
	peers := rt.NearestPeers(id, 1)
	if len(peers) > 0 {
		return peers[0]
	}

	//log.Debugf("NearestPeer: Returning nil, table size = %d", rt.Size())
	return ""
}

// NearestPeers returns a list of the 'count' closest peers to the given ID
func (rt *RoutingTable) NearestPeers(id ID, count int) []peer.ID {
	// This is the number of bits _we_ share with the key. All peers in this
	// bucket share cpl bits with us and will therefore share at least cpl+1
	// bits with the given key. +1 because both the target and all peers in
	// this bucket differ from us in the cpl bit.
	cpl := CommonPrefixLen(id, rt.local)

	// It's assumed that this also protects the buckets.
	rt.tabLock.RLock()

	// Get bucket index or last bucket
	if cpl >= len(rt.buckets) {
		cpl = len(rt.buckets) - 1
	}
	//Added by Kanemitsu
	if(rt.isKadRTT){
		count = rt.buckets[cpl].beta
	}

	pds := peerDistanceSorter{
		peers:  make([]peerDistance, 0, count+rt.bucketsize),
		target: id,
	}


	// Add peers from the target bucket (cpl+1 shared bits).
	pds.appendPeersFromList(rt.buckets[cpl].list)

	// If we're short, add peers from all buckets to the right. All buckets
	// to the right share exactly cpl bits (as opposed to the cpl+1 bits
	// shared by the peers in the cpl bucket).
	//
	// This is, unfortunately, less efficient than we'd like. We will switch
	// to a trie implementation eventually which will allow us to find the
	// closest N peers to any target key.

	if pds.Len() < count {
		for i := cpl + 1; i < len(rt.buckets); i++ {
			pds.appendPeersFromList(rt.buckets[i].list)
		}
	}

	// If we're still short, add in buckets that share _fewer_ bits. We can
	// do this bucket by bucket because each bucket will share 1 fewer bit
	// than the last.
	//
	// * bucket cpl-1: cpl-1 shared bits.
	// * bucket cpl-2: cpl-2 shared bits.
	// ...
	for i := cpl - 1; i >= 0 && pds.Len() < count; i-- {
		pds.appendPeersFromList(rt.buckets[i].list)
	}
	//rt.tabLock.RUnlock()

	// Sort by distance to local peer
	pds.sort()

	if count < pds.Len() {
		pds.peers = pds.peers[:count]
	}

	if(rt.isKadRTT && len(pds.peers)>0){
		//get the 1st entry in pds that has the shortest distance with ID.
		first := pds.peers[0]
		fID := first.p
		fcpl := rt.bucketIdForPeer(fID)
		fRTT := rt.buckets[fcpl].getPeer(fID).rtt

		minDist := first.distance

		rttpds := peerRTTDistanceSorter{
			peers:  make([]peerRTTDistance, 0, count+rt.bucketsize),
			target: id,
		}
		rttpds.appendPeersFromList(rt.buckets[cpl].list)

		if rttpds.Len() < count {
			for i := cpl + 1; i < len(rt.buckets); i++ {
				rttpds.appendPeersFromList(rt.buckets[i].list)
			}
		}

		// If we're still short, add in buckets that share _fewer_ bits. We can
		// do this bucket by bucket because each bucket will share 1 fewer bit
		// than the last.
		//
		// * bucket cpl-1: cpl-1 shared bits.
		// * bucket cpl-2: cpl-2 shared bits.
		// ...
		for i := cpl - 1; i >= 0 && rttpds.Len() < count; i-- {
			rttpds.appendPeersFromList(rt.buckets[i].list)
		}
		//Sort by increasing order of RTT.
		rttpds.sort()

		rt.tabLock.RUnlock()
		//For each entry of rttpds, the log condition is checked.
		//if the condition is satisfied, the entry is added with hither
		//priority.
		OKList := make([]peer.ID, 0, pds.Len())
		NGList := make([]peer.ID, 0, pds.Len())

		for _,p := range rttpds.peers{
			minDistInt := big.NewInt(0).SetBytes(minDist)
			pDistInt := big.NewInt(0).SetBytes(p.distance)
			minDistIntDouble := minDistInt.Mul(minDistInt,big.NewInt(2))
			if(pDistInt.Cmp(minDistIntDouble) < 0){
				//If it's OK, p should have higher priority.
				rtt := p.rtt
				if(rtt <= fRTT){
					OKList = append(OKList, p.p)

				}else{
					NGList = append(NGList,p.p)
				}


			}else{
				NGList = append(NGList,p.p)
			}

		}
		out := make([]peer.ID, 0, pds.Len())
		for _, p := range OKList {
			out = append(out, p)
		}

		for _, p := range NGList {
			out = append(out, p)
		}
		return out


	}else{
		rt.tabLock.RUnlock()
		out := make([]peer.ID, 0, pds.Len())
		for _, p := range pds.peers {
			out = append(out, p.p)
		}


		return out
	}

}

// Size returns the total number of peers in the routing table
func (rt *RoutingTable) Size() int {
	var tot int
	rt.tabLock.RLock()
	for _, buck := range rt.buckets {
		tot += buck.len()
	}
	rt.tabLock.RUnlock()
	return tot
}

// ListPeers takes a RoutingTable and returns a list of all peers from all buckets in the table.
func (rt *RoutingTable) ListPeers() []peer.ID {
	rt.tabLock.RLock()
	defer rt.tabLock.RUnlock()

	var peers []peer.ID
	for _, buck := range rt.buckets {
		peers = append(peers, buck.peerIds()...)
	}
	return peers
}

// Print prints a descriptive statement about the provided RoutingTable
func (rt *RoutingTable) Print() {
	fmt.Printf("Routing Table, bs = %d, Max latency = %d\n", rt.bucketsize, rt.maxLatency)
	rt.tabLock.RLock()

	for i, b := range rt.buckets {
		fmt.Printf("\tbucket: %d\n", i)

		for e := b.list.Front(); e != nil; e = e.Next() {
			p := e.Value.(*PeerInfo).Id
			fmt.Printf("\t\t- %s %s\n", p.Pretty(), rt.metrics.LatencyEWMA(p).String())
		}
	}
	rt.tabLock.RUnlock()
}

// GetDiversityStats returns the diversity stats for the Routing Table if a diversity Filter
// is configured.
func (rt *RoutingTable) GetDiversityStats() []peerdiversity.CplDiversityStats {
	if rt.df != nil {
		return rt.df.GetDiversityStats()
	}
	return nil
}

// the caller is responsible for the locking
func (rt *RoutingTable) bucketIdForPeer(p peer.ID) int {
	peerID := ConvertPeerID(p)
	cpl := CommonPrefixLen(peerID, rt.local)
	bucketID := cpl
	if bucketID >= len(rt.buckets) {
		bucketID = len(rt.buckets) - 1
	}
	return bucketID
}

// maxCommonPrefix returns the maximum common prefix length between any peer in
// the table and the current peer.
func (rt *RoutingTable) maxCommonPrefix() uint {
	rt.tabLock.RLock()
	defer rt.tabLock.RUnlock()

	for i := len(rt.buckets) - 1; i >= 0; i-- {
		if rt.buckets[i].len() > 0 {
			return rt.buckets[i].maxCommonPrefix(rt.local)
		}
	}
	return 0
}
