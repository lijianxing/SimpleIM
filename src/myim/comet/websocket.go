package main

import (
	"io"
	"myim/libs/bytes"
	"myim/libs/define"
	"myim/libs/net/websocket"
	"myim/libs/proto"
	itime "myim/libs/time"
	"net"
	"time"

	log "github.com/thinkboy/log4go"
)

// InitWebsocket listen all tcp.bind and start accept connections.
func InitWebsocket(addrs []string, accept int) (err error) {
	var (
		bind     string
		listener *net.TCPListener
		addr     *net.TCPAddr
	)
	for _, bind = range addrs {
		if addr, err = net.ResolveTCPAddr("tcp4", bind); err != nil {
			log.Error("net.ResolveTCPAddr(\"tcp4\", \"%s\") error(%v)", bind, err)
			return
		}
		if listener, err = net.ListenTCP("tcp4", addr); err != nil {
			log.Error("net.ListenTCP(\"tcp4\", \"%s\") error(%v)", bind, err)
			return
		}
		log.Info("start ws listen: \"%s\"", bind)
		// split N core accept
		for i := 0; i < accept; i++ {
			go acceptWebsocket(DefaultServer, listener)
		}
	}
	return
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.  Accept blocks; the caller typically
// invokes it in a go statement.
func acceptWebsocket(server *Server, lis *net.TCPListener) {
	var (
		conn *net.TCPConn
		err  error
		r    int
	)
	for {
		if conn, err = lis.AcceptTCP(); err != nil {
			// if listener close then return
			log.Error("listener.Accept(\"%s\") error(%v)", lis.Addr().String(), err)
			return
		}
		if err = conn.SetKeepAlive(server.Options.TCPKeepalive); err != nil {
			log.Error("conn.SetKeepAlive() error(%v)", err)
			return
		}
		if err = conn.SetReadBuffer(server.Options.TCPRcvbuf); err != nil {
			log.Error("conn.SetReadBuffer() error(%v)", err)
			return
		}
		if err = conn.SetWriteBuffer(server.Options.TCPSndbuf); err != nil {
			log.Error("conn.SetWriteBuffer() error(%v)", err)
			return
		}
		go serveWebsocket(server, conn, r)
		if r++; r == maxInt {
			r = 0
		}
	}
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.  Accept blocks; the caller typically
// invokes it in a go statement.
func acceptWebsocketWithTLS(server *Server, lis net.Listener) {
	var (
		conn net.Conn
		err  error
		r    int
	)
	for {
		if conn, err = lis.Accept(); err != nil {
			// if listener close then return
			log.Error("listener.Accept(\"%s\") error(%v)", lis.Addr().String(), err)
			return
		}
		go serveWebsocket(server, conn, r)
		if r++; r == maxInt {
			r = 0
		}
	}
}

func serveWebsocket(server *Server, conn net.Conn, r int) {
	var (
		// timer
		tr = server.round.Timer(r)
		rp = server.round.Reader(r)
		wp = server.round.Writer(r)
	)
	if Debug {
		// ip addr
		lAddr := conn.LocalAddr().String()
		rAddr := conn.RemoteAddr().String()
		log.Debug("start ws serve \"%s\" with \"%s\"", lAddr, rAddr)
	}
	server.serveWebsocket(conn, rp, wp, tr)
}

// TODO linger close?
func (server *Server) serveWebsocket(conn net.Conn, rp, wp *bytes.Pool, tr *itime.Timer) {
	var (
		err error
		key string
		hb  time.Duration // heartbeat
		p   *proto.Proto
		b   *Bucket
		trd *itime.TimerData
		rb  = rp.Get()
		ch  = NewChannel(server.Options.CliProto, server.Options.SvrProto)
		rr  = &ch.Reader
		wr  = &ch.Writer
		ws  *websocket.Conn // websocket
		req *websocket.Request
	)
	// reader
	ch.Reader.ResetBuffer(conn, rb.Bytes())
	// handshake
	trd = tr.Add(server.Options.HandshakeTimeout, func() {
		conn.Close()
	})
	// websocket
	if req, err = websocket.ReadRequest(rr); err != nil || req.RequestURI != "/sub" {
		conn.Close()
		tr.Del(trd)
		rp.Put(rb)
		if err != io.EOF {
			log.Error("http.ReadRequest(rr) error(%v)", err)
		}
		return
	}
	// writer
	wb := wp.Get()
	ch.Writer.ResetBuffer(conn, wb.Bytes())
	if ws, err = websocket.Upgrade(conn, rr, wr, req); err != nil {
		conn.Close()
		tr.Del(trd)
		rp.Put(rb)
		wp.Put(wb)
		if err != io.EOF {
			log.Error("websocket.NewServerConn error(%v)", err)
		}
		return
	}
	// must not setadv, only used in auth
	if p, err = ch.CliProto.Set(); err == nil {
		if key, hb, err = server.authWebsocket(ws, p); err == nil {
			b = server.Bucket(key)
			err = b.Put(key, ch)
		}
	}
	if err != nil {
		ws.Close()
		rp.Put(rb)
		wp.Put(wb)
		tr.Del(trd)
		if err != io.EOF && err != websocket.ErrMessageClose {
			log.Error("key: %s handshake failed error(%v)", key, err)
		}
		return
	}
	trd.Key = key
	tr.Set(trd, hb)

	// increase ws stat
	server.Stat.IncrWsOnline()

	// hanshake ok start dispatch goroutine
	go server.dispatchWebsocket(key, ws, wp, wb, ch)
	for {
		if p, err = ch.CliProto.Set(); err != nil {
			break
		}
		if err = p.ReadWebsocket(ws); err != nil {
			break
		}

		if p.Operation == define.OP_LOGOUT {
			// only break loop
			break
		} else {
			tr.Set(trd, hb) // 当成hb
			if err = server.operator.Operate(key, p); err != nil {
				log.Error("operate msg failed.key:%s, msg:%s", key, p)
				continue // 忽略错误
			}
		}
		ch.CliProto.SetAdv()
		ch.Signal()
	}
	if err != nil && err != io.EOF && err != websocket.ErrMessageClose {
		log.Error("key: %s server tcp failed error(%v)", key, err)
	}
	b.Del(key)
	tr.Del(trd)
	ws.Close()
	ch.Close()
	rp.Put(rb)
	if err = server.operator.Disconnect(key); err != nil {
		log.Error("key: %s operator do disconnect error(%v)", key, err)
	}
	if Debug {
		log.Debug("key: %s server tcp goroutine exit", key)
	}
	// decrease ws stat
	server.Stat.DecrWsOnline()
	return
}

// dispatch accepts connections on the listener and serves requests
// for each incoming connection.  dispatch blocks; the caller typically
// invokes it in a go statement.
func (server *Server) dispatchWebsocket(key string, ws *websocket.Conn, wp *bytes.Pool, wb *bytes.Buffer, ch *Channel) {
	var (
		err    error
		finish bool
	)
	if Debug {
		log.Debug("key: %s start dispatch tcp goroutine", key)
	}
	for {
		var p = ch.Ready()
		if Debug {
			log.Debug("key:%s dispatch msg:%s", key, p.Body)
		}

		closeConn := false
		switch p {
		case proto.ProtoFinish:
			if Debug {
				log.Debug("key: %s wakeup exit dispatch goroutine", key)
			}
			finish = true
			goto failed
		case proto.ProtoReady:
			// fetch message from svrbox(client send)
			for {
				if p, err = ch.CliProto.Get(); err != nil {
					err = nil // must be empty error
					break
				}
				if p.Operation != define.OP_NONE {
					if err = p.WriteWebsocket(ws); err != nil {
						goto failed
					}
				}
				p.Body = nil // avoid memory leak
				ch.CliProto.GetAdv()
			}
		default:
			// 踢人
			if p.Operation == define.OP_KICKOUT {
				log.Warn("kickout user link %s", key)
				closeConn = true
			}
			// server send
			if err = p.WriteWebsocket(ws); err != nil {
				goto failed
			}
		}
		// only hungry flush response
		if err = ws.Flush(); err != nil {
			break
		}

		if closeConn {
			break
		}
	}
failed:
	if err != nil && err != io.EOF && err != websocket.ErrMessageClose {
		log.Error("key: %s dispatch tcp error(%v)", key, err)
	}
	ws.Close()
	wp.Put(wb)
	// must ensure all channel message discard, for reader won't blocking Signal
	for !finish {
		finish = (ch.Ready() == proto.ProtoFinish)
	}
	if Debug {
		log.Debug("key: %s dispatch goroutine exit", key)
	}
	return
}

// auth for goim handshake with client, use rsa & aes.
func (server *Server) authWebsocket(ws *websocket.Conn, p *proto.Proto) (key string, heartbeat time.Duration, err error) {
	if err = p.ReadWebsocket(ws); err != nil {
		return
	}
	if p.Operation != define.OP_LOGIN {
		err = ErrOperation
		return
	}
	if key, heartbeat, err = server.operator.Connect(p); err != nil {
		return
	}
	p.Body = nil
	p.Operation = define.OP_LOGIN_REPLY
	if err = p.WriteWebsocket(ws); err != nil {
		return
	}
	err = ws.Flush()
	return
}
