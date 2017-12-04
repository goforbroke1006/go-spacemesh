package conn

import (
	"context"
	"io"
	"net"
	"time"

	logging "gx/ipfs/QmSpJByNKFX1sCsHBEp3R73FL4NF6FnQTEGyNAXHm2GS52/go-log"
	lgbl "gx/ipfs/QmT4PgCNdv73hnFAqzHqwW44q7M9PWpykSswHDxndquZbc/go-libp2p-loggables"
	mpool "gx/ipfs/QmWBug6eBS7AxRdCDVuSY5CnSit7cS2XnPFYJWqWDumhCG/go-msgio/mpool"
	ma "gx/ipfs/QmXY77cVe7rVRQXZZQRioukUM7aRW3BTcAgJe12MCtb3Ji/go-multiaddr"
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	iconn "gx/ipfs/QmZD7kdgd6eiBCvNbyvsFQ11h3ainoCJpJRbecDx3CPcbZ/go-libp2p-interface-conn"
	ic "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
	tpt "gx/ipfs/Qme2XMfKbWzzYd92YvA1qnFMe3pGDR86j5BcFtx4PwdRvr/go-libp2p-transport"
)

var log = logging.Logger("conn")

// ReleaseBuffer puts the given byte array back into the buffer pool,
// first verifying that it is the correct size
func ReleaseBuffer(b []byte) {
	log.Debugf("Releasing buffer! (cap,size = %d, %d)", cap(b), len(b))
	mpool.ByteSlicePool.Put(uint32(cap(b)), b)
}

// singleConn represents a single connection to another Peer (IPFS Node).
type singleConn struct {
	local  peer.ID
	remote peer.ID
	maconn tpt.Conn
	event  io.Closer
}

// newConn constructs a new connection
func newSingleConn(ctx context.Context, local, remote peer.ID, maconn tpt.Conn) iconn.Conn {
	ml := lgbl.Dial("conn", local, remote, maconn.LocalMultiaddr(), maconn.RemoteMultiaddr())

	conn := &singleConn{
		local:  local,
		remote: remote,
		maconn: maconn,
		event:  log.EventBegin(ctx, "connLifetime", ml),
	}

	log.Debugf("newSingleConn %p: %v to %v", conn, local, remote)
	return conn
}

// close is the internal close function, called by ContextCloser.Close
func (c *singleConn) Close() error {
	defer func() {
		if c.event != nil {
			c.event.Close()
			c.event = nil
		}
	}()

	// close underlying connection
	return c.maconn.Close()
}

// ID is an identifier unique to this connection.
func (c *singleConn) ID() string {
	return iconn.ID(c)
}

func (c *singleConn) String() string {
	return iconn.String(c, "singleConn")
}

func (c *singleConn) LocalAddr() net.Addr {
	return c.maconn.LocalAddr()
}

func (c *singleConn) RemoteAddr() net.Addr {
	return c.maconn.RemoteAddr()
}

func (c *singleConn) LocalPrivateKey() ic.PrivKey {
	return nil
}

func (c *singleConn) RemotePublicKey() ic.PubKey {
	return nil
}

func (c *singleConn) SetDeadline(t time.Time) error {
	return c.maconn.SetDeadline(t)
}
func (c *singleConn) SetReadDeadline(t time.Time) error {
	return c.maconn.SetReadDeadline(t)
}

func (c *singleConn) SetWriteDeadline(t time.Time) error {
	return c.maconn.SetWriteDeadline(t)
}

// LocalMultiaddr is the Multiaddr on this side
func (c *singleConn) LocalMultiaddr() ma.Multiaddr {
	return c.maconn.LocalMultiaddr()
}

// RemoteMultiaddr is the Multiaddr on the remote side
func (c *singleConn) RemoteMultiaddr() ma.Multiaddr {
	return c.maconn.RemoteMultiaddr()
}

func (c *singleConn) Transport() tpt.Transport {
	return c.maconn.Transport()
}

// LocalPeer is the Peer on this side
func (c *singleConn) LocalPeer() peer.ID {
	return c.local
}

// RemotePeer is the Peer on the remote side
func (c *singleConn) RemotePeer() peer.ID {
	return c.remote
}

// Read reads data, net.Conn style
func (c *singleConn) Read(buf []byte) (int, error) {
	return c.maconn.Read(buf)
}

// Write writes data, net.Conn style
func (c *singleConn) Write(buf []byte) (int, error) {
	return c.maconn.Write(buf)
}
