package identify

import (
	"context"
	"strings"
	"sync"

	pb "github.com/libp2p/go-libp2p/p2p/protocol/identify/pb"

	pstore "gx/ipfs/QmPgDWmTmuzvP7QE5zwo1TmjbJme9pmZHNujB2453jkCTr/go-libp2p-peerstore"
	host "gx/ipfs/QmRS46AyqtpJBsf1zmQdeizSDEzo1qkWR7rdEuPFAv8237/go-libp2p-host"
	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	lgbl "gx/ipfs/QmT4PgCNdv73hnFAqzHqwW44q7M9PWpykSswHDxndquZbc/go-libp2p-loggables"
	msmux "gx/ipfs/QmTnsezaB1wWNRHeHnYrm8K4d5i9wtyj3GsqjC3Rt5b5v5/go-multistream"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	ggio "gx/ipfs/QmZ4Qi3GaRbjcx28Sme5eMH7RQjGkt8wHxt2a65oLaeFEV/gogo-protobuf/io"
	ic "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	inet "gx/ipfs/QmbD5yKbXahNvoMqzeuNyKQA9vAs9fUvJg2GXeWU1fVqY5/go-libp2p-net"
	metrics "gx/ipfs/QmbXmeK6KgUAkbyVGRxXknupmWAHnt6ryghT8BFSsEh2sB/go-libp2p-metrics"
	mstream "gx/ipfs/QmbXmeK6KgUAkbyVGRxXknupmWAHnt6ryghT8BFSsEh2sB/go-libp2p-metrics/stream"
	semver "gx/ipfs/QmcrrEpx3VMUbrbgVroH3YiYyUS5c4YAykzyPJWKspUYLa/go-semver/semver"
)

var log = logging.Logger("net/identify")

// ID is the protocol.ID of the Identify Service.
const ID = "/ipfs/id/1.0.0"

// LibP2PVersion holds the current protocol version for a client running this code
// TODO(jbenet): fix the versioning mess.
const LibP2PVersion = "ipfs/0.1.0"

var ClientVersion = "go-libp2p/3.3.4"

// IDService is a structure that implements ProtocolIdentify.
// It is a trivial service that gives the other peer some
// useful information about the local peer. A sort of hello.
//
// The IDService sends:
//  * Our IPFS Protocol Version
//  * Our IPFS Agent Version
//  * Our public Listen Addresses
type IDService struct {
	Host host.Host

	Reporter metrics.Reporter
	// connections undergoing identification
	// for wait purposes
	currid map[inet.Conn]chan struct{}
	currmu sync.RWMutex

	// our own observed addresses.
	// TODO: instead of expiring, remove these when we disconnect
	observedAddrs ObservedAddrSet
}

// NewIDService constructs a new *IDService and activates it by
// attaching its stream handler to the given host.Host.
func NewIDService(h host.Host) *IDService {
	s := &IDService{
		Host:   h,
		currid: make(map[inet.Conn]chan struct{}),
	}
	h.SetStreamHandler(ID, s.RequestHandler)
	return s
}

// OwnObservedAddrs returns the addresses peers have reported we've dialed from
func (ids *IDService) OwnObservedAddrs() []ma.Multiaddr {
	return ids.observedAddrs.Addrs()
}

func (ids *IDService) IdentifyConn(c inet.Conn) {
	ids.currmu.Lock()
	if wait, found := ids.currid[c]; found {
		ids.currmu.Unlock()
		log.Debugf("IdentifyConn called twice on: %s", c)
		<-wait // already identifying it. wait for it.
		return
	}
	ch := make(chan struct{})
	ids.currid[c] = ch
	ids.currmu.Unlock()

	defer close(ch)

	s, err := c.NewStream()
	if err != nil {
		log.Debugf("error opening initial stream for %s: %s", ID, err)
		log.Event(context.TODO(), "IdentifyOpenFailed", c.RemotePeer())
		c.Close()
		return
	}
	defer s.Close()

	s.SetProtocol(ID)

	if ids.Reporter != nil {
		s = mstream.WrapStream(s, ids.Reporter)
	}

	// ok give the response to our handler.
	if err := msmux.SelectProtoOrFail(ID, s); err != nil {
		log.Event(context.TODO(), "IdentifyOpenFailed", c.RemotePeer(), logging.Metadata{"error": err})
		return
	}

	ids.ResponseHandler(s)

	ids.currmu.Lock()
	_, found := ids.currid[c]
	delete(ids.currid, c)
	ids.currmu.Unlock()

	if !found {
		log.Errorf("IdentifyConn failed to find channel (programmer error) for %s", c)
		return
	}
}

func (ids *IDService) RequestHandler(s inet.Stream) {
	defer s.Close()
	c := s.Conn()

	if ids.Reporter != nil {
		s = mstream.WrapStream(s, ids.Reporter)
	}

	w := ggio.NewDelimitedWriter(s)
	mes := pb.Identify{}
	ids.populateMessage(&mes, s.Conn())
	w.WriteMsg(&mes)

	log.Debugf("%s sent message to %s %s", ID,
		c.RemotePeer(), c.RemoteMultiaddr())
}

func (ids *IDService) ResponseHandler(s inet.Stream) {
	defer s.Close()
	c := s.Conn()

	r := ggio.NewDelimitedReader(s, 2048)
	mes := pb.Identify{}
	if err := r.ReadMsg(&mes); err != nil {
		log.Warning("error reading identify message: ", err)
		return
	}
	ids.consumeMessage(&mes, c)

	log.Debugf("%s received message from %s %s", ID,
		c.RemotePeer(), c.RemoteMultiaddr())
}

func (ids *IDService) populateMessage(mes *pb.Identify, c inet.Conn) {

	// set protocols this node is currently handling
	protos := ids.Host.Mux().Protocols()
	mes.Protocols = make([]string, len(protos))
	for i, p := range protos {
		mes.Protocols[i] = string(p)
	}

	// observed address so other side is informed of their
	// "public" address, at least in relation to us.
	mes.ObservedAddr = c.RemoteMultiaddr().Bytes()

	// set listen addrs, get our latest addrs from Host.
	laddrs := ids.Host.Addrs()
	mes.ListenAddrs = make([][]byte, len(laddrs))
	for i, addr := range laddrs {
		mes.ListenAddrs[i] = addr.Bytes()
	}
	log.Debugf("%s sent listen addrs to %s: %s", c.LocalPeer(), c.RemotePeer(), laddrs)

	// set our public key
	ownKey := ids.Host.Peerstore().PubKey(ids.Host.ID())
	if ownKey == nil {
		log.Errorf("did not have own public key in Peerstore")
	} else {
		if kb, err := ownKey.Bytes(); err != nil {
			log.Errorf("failed to convert key to bytes")
		} else {
			mes.PublicKey = kb
		}
	}

	// set protocol versions
	pv := LibP2PVersion
	av := ClientVersion
	mes.ProtocolVersion = &pv
	mes.AgentVersion = &av
}

func (ids *IDService) consumeMessage(mes *pb.Identify, c inet.Conn) {
	p := c.RemotePeer()

	// mes.Protocols
	ids.Host.Peerstore().SetProtocols(p, mes.Protocols...)

	// mes.ObservedAddr
	ids.consumeObservedAddress(mes.GetObservedAddr(), c)

	// mes.ListenAddrs
	laddrs := mes.GetListenAddrs()
	lmaddrs := make([]ma.Multiaddr, 0, len(laddrs))
	for _, addr := range laddrs {
		maddr, err := ma.NewMultiaddrBytes(addr)
		if err != nil {
			log.Debugf("%s failed to parse multiaddr from %s %s", ID,
				p, c.RemoteMultiaddr())
			continue
		}
		lmaddrs = append(lmaddrs, maddr)
	}

	// if the address reported by the connection roughly matches their annoucned
	// listener addresses, its likely to be an external NAT address
	if HasConsistentTransport(c.RemoteMultiaddr(), lmaddrs) {
		lmaddrs = append(lmaddrs, c.RemoteMultiaddr())
	}

	// update our peerstore with the addresses. here, we SET the addresses, clearing old ones.
	// We are receiving from the peer itself. this is current address ground truth.
	ids.Host.Peerstore().SetAddrs(p, lmaddrs, pstore.ConnectedAddrTTL)
	log.Debugf("%s received listen addrs for %s: %s", c.LocalPeer(), c.RemotePeer(), lmaddrs)

	// get protocol versions
	pv := mes.GetProtocolVersion()
	av := mes.GetAgentVersion()

	// version check. if we shouldn't talk, bail.
	// TODO: at this point, we've already exchanged information.
	// move this into a first handshake before the connection can open streams.
	if !protocolVersionsAreCompatible(pv, LibP2PVersion) {
		logProtocolMismatchDisconnect(c, pv, av)
		c.Close()
		return
	}

	ids.Host.Peerstore().Put(p, "ProtocolVersion", pv)
	ids.Host.Peerstore().Put(p, "AgentVersion", av)

	// get the key from the other side. we may not have it (no-auth transport)
	ids.consumeReceivedPubKey(c, mes.PublicKey)
}

func (ids *IDService) consumeReceivedPubKey(c inet.Conn, kb []byte) {
	lp := c.LocalPeer()
	rp := c.RemotePeer()

	if kb == nil {
		log.Debugf("%s did not receive public key for remote peer: %s", lp, rp)
		return
	}

	newKey, err := ic.UnmarshalPublicKey(kb)
	if err != nil {
		log.Errorf("%s cannot unmarshal key from remote peer: %s", lp, rp)
		return
	}

	// verify key matches peer.ID
	np, err := peer.IDFromPublicKey(newKey)
	if err != nil {
		log.Debugf("%s cannot get peer.ID from key of remote peer: %s, %s", lp, rp, err)
		return
	}

	if np != rp {
		// if the newKey's peer.ID does not match known peer.ID...

		if rp == "" && np != "" {
			// if local peerid is empty, then use the new, sent key.
			err := ids.Host.Peerstore().AddPubKey(rp, newKey)
			if err != nil {
				log.Debugf("%s could not add key for %s to peerstore: %s", lp, rp, err)
			}

		} else {
			// we have a local peer.ID and it does not match the sent key... error.
			log.Errorf("%s received key for remote peer %s mismatch: %s", lp, rp, np)
		}
		return
	}

	currKey := ids.Host.Peerstore().PubKey(rp)
	if currKey == nil {
		// no key? no auth transport. set this one.
		err := ids.Host.Peerstore().AddPubKey(rp, newKey)
		if err != nil {
			log.Debugf("%s could not add key for %s to peerstore: %s", lp, rp, err)
		}
		return
	}

	// ok, we have a local key, we should verify they match.
	if currKey.Equals(newKey) {
		return // ok great. we're done.
	}

	// weird, got a different key... but the different key MATCHES the peer.ID.
	// this odd. let's log error and investigate. this should basically never happen
	// and it means we have something funky going on and possibly a bug.
	log.Errorf("%s identify got a different key for: %s", lp, rp)

	// okay... does ours NOT match the remote peer.ID?
	cp, err := peer.IDFromPublicKey(currKey)
	if err != nil {
		log.Errorf("%s cannot get peer.ID from local key of remote peer: %s, %s", lp, rp, err)
		return
	}
	if cp != rp {
		log.Errorf("%s local key for remote peer %s yields different peer.ID: %s", lp, rp, cp)
		return
	}

	// okay... curr key DOES NOT match new key. both match peer.ID. wat?
	log.Errorf("%s local key and received key for %s do not match, but match peer.ID", lp, rp)
}

// HasConsistentTransport returns true if the address 'a' shares a
// protocol set with any address in the green set. This is used
// to check if a given address might be one of the addresses a peer is
// listening on.
func HasConsistentTransport(a ma.Multiaddr, green []ma.Multiaddr) bool {
	protosMatch := func(a, b []ma.Protocol) bool {
		if len(a) != len(b) {
			return false
		}

		for i, p := range a {
			if b[i].Code != p.Code {
				return false
			}
		}
		return true
	}

	protos := a.Protocols()

	for _, ga := range green {
		if protosMatch(protos, ga.Protocols()) {
			return true
		}
	}

	return false
}

// IdentifyWait returns a channel which will be closed once
// "ProtocolIdentify" (handshake3) finishes on given conn.
// This happens async so the connection can start to be used
// even if handshake3 knowledge is not necesary.
// Users **MUST** call IdentifyWait _after_ IdentifyConn
func (ids *IDService) IdentifyWait(c inet.Conn) <-chan struct{} {
	ids.currmu.Lock()
	ch, found := ids.currid[c]
	ids.currmu.Unlock()
	if found {
		return ch
	}

	// if not found, it means we are already done identifying it, or
	// haven't even started. either way, return a new channel closed.
	ch = make(chan struct{})
	close(ch)
	return ch
}

func (ids *IDService) consumeObservedAddress(observed []byte, c inet.Conn) {
	if observed == nil {
		return
	}

	maddr, err := ma.NewMultiaddrBytes(observed)
	if err != nil {
		log.Debugf("error parsing received observed addr for %s: %s", c, err)
		return
	}

	// we should only use ObservedAddr when our connection's LocalAddr is one
	// of our ListenAddrs. If we Dial out using an ephemeral addr, knowing that
	// address's external mapping is not very useful because the port will not be
	// the same as the listen addr.
	ifaceaddrs, err := ids.Host.Network().InterfaceListenAddresses()
	if err != nil {
		log.Infof("failed to get interface listen addrs", err)
		return
	}

	log.Debugf("identify identifying observed multiaddr: %s %s", c.LocalMultiaddr(), ifaceaddrs)
	if !addrInAddrs(c.LocalMultiaddr(), ifaceaddrs) {
		// not in our list
		return
	}

	// ok! we have the observed version of one of our ListenAddresses!
	log.Debugf("added own observed listen addr: %s --> %s", c.LocalMultiaddr(), maddr)
	ids.observedAddrs.Add(maddr, c.RemoteMultiaddr())
}

func addrInAddrs(a ma.Multiaddr, as []ma.Multiaddr) bool {
	for _, b := range as {
		if a.Equal(b) {
			return true
		}
	}
	return false
}

// protocolVersionsAreCompatible checks that the two implementations
// can talk to each other. It will use semver, but for now while
// we're in tight development, we will return false for minor version
// changes too.
func protocolVersionsAreCompatible(v1, v2 string) bool {
	if strings.HasPrefix(v1, "ipfs/") {
		v1 = v1[5:]
	}
	if strings.HasPrefix(v2, "ipfs/") {
		v2 = v2[5:]
	}

	v1s, err := semver.NewVersion(v1)
	if err != nil {
		return false
	}

	v2s, err := semver.NewVersion(v2)
	if err != nil {
		return false
	}

	return v1s.Major == v2s.Major && v1s.Minor == v2s.Minor
}

// netNotifiee defines methods to be used with the IpfsDHT
type netNotifiee IDService

func (nn *netNotifiee) IDService() *IDService {
	return (*IDService)(nn)
}

func (nn *netNotifiee) Connected(n inet.Network, v inet.Conn) {
	// TODO: deprecate the setConnHandler hook, and kick off
	// identification here.
}

func (nn *netNotifiee) Disconnected(n inet.Network, v inet.Conn) {
	// undo the setting of addresses to peer.ConnectedAddrTTL we did
	ids := nn.IDService()
	ps := ids.Host.Peerstore()
	addrs := ps.Addrs(v.RemotePeer())
	ps.SetAddrs(v.RemotePeer(), addrs, pstore.RecentlyConnectedAddrTTL)
}

func (nn *netNotifiee) OpenedStream(n inet.Network, v inet.Stream) {}
func (nn *netNotifiee) ClosedStream(n inet.Network, v inet.Stream) {}
func (nn *netNotifiee) Listen(n inet.Network, a ma.Multiaddr)      {}
func (nn *netNotifiee) ListenClose(n inet.Network, a ma.Multiaddr) {}

func logProtocolMismatchDisconnect(c inet.Conn, protocol, agent string) {
	lm := make(lgbl.DeferredMap)
	lm["remotePeer"] = func() interface{} { return c.RemotePeer().Pretty() }
	lm["remoteAddr"] = func() interface{} { return c.RemoteMultiaddr().String() }
	lm["protocolVersion"] = protocol
	lm["agentVersion"] = agent
	log.Event(context.TODO(), "IdentifyProtocolMismatch", lm)
	log.Debugf("IdentifyProtocolMismatch %s %s %s (disconnected)", c.RemotePeer(), protocol, agent)
}
