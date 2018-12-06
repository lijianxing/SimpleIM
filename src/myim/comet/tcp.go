package main

import (
	"io"
	"myim/libs/bufio"
	"myim/libs/bytes"
	"myim/libs/define"
	"myim/libs/proto"
	itime "myim/libs/time"
	"net"
	"time"

	log "github.com/thinkboy/log4go"
)

// InitTCP listen all tcp.bind and start accept connections.
func InitTCP(addrs []string, accept int) (err error) {
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
		log.Info("start tcp listen: \"%s\"", bind)
		// split N core accept
		for i := 0; i < accept; i++ {
			// 每个addr一个goroutine
			go acceptTCP(DefaultServer, listener)
		}
	}
	return
}

// Accept accepts connections on the listener and serves requests
// for each incoming connection.  Accept blocks; the caller typically
// invokes it in a go statement.
func acceptTCP(server *Server, lis *net.TCPListener) {
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

		// 默认是false
		if err = conn.SetKeepAlive(server.Options.TCPKeepalive); err != nil {
			log.Error("conn.SetKeepAlive() error(%v)", err)
			return
		}

		// 默认1024
		if err = conn.SetReadBuffer(server.Options.TCPRcvbuf); err != nil {
			log.Error("conn.SetReadBuffer() error(%v)", err)
			return
		}
		// 默认1024
		if err = conn.SetWriteBuffer(server.Options.TCPSndbuf); err != nil {
			log.Error("conn.SetWriteBuffer() error(%v)", err)
			return
		}

		// 每连接一个goroutine
		go serveTCP(server, conn, r)
		if r++; r == maxInt {
			r = 0
		}
	}
}

func serveTCP(server *Server, conn *net.TCPConn, r int) {
	var (
		// timer
		tr = server.round.Timer(r)
		rp = server.round.Reader(r)
		wp = server.round.Writer(r)
		// ip addr
		lAddr = conn.LocalAddr().String()
		rAddr = conn.RemoteAddr().String()
	)
	if Debug {
		log.Debug("start tcp serve \"%s\" with \"%s\"", lAddr, rAddr)
	}
	server.serveTCP(conn, rp, wp, tr)
}

// TODO linger close?
func (server *Server) serveTCP(conn *net.TCPConn, rp, wp *bytes.Pool, tr *itime.Timer) {
	var (
		err error
		key string
		hb  time.Duration // heartbeat
		p   *proto.Proto
		b   *Bucket
		trd *itime.TimerData
		rb  = rp.Get()
		wb  = wp.Get()
		ch  = NewChannel(server.Options.CliProto, server.Options.SvrProto)
		rr  = &ch.Reader
		wr  = &ch.Writer
	)
	// 关联channel的r/w buffer到conn
	ch.Reader.ResetBuffer(conn, rb.Bytes())
	ch.Writer.ResetBuffer(conn, wb.Bytes())

	// handshake (建立连接后不发数据的超时)
	trd = tr.Add(server.Options.HandshakeTimeout, func() {
		conn.Close()
	})

	// must not setadv, only used in auth
	if p, err = ch.CliProto.Set(); err == nil {
		if key, hb, err = server.authTCP(rr, wr, p); err == nil {
			b = server.Bucket(key)
			err = b.Put(key, ch)
		}
	}
	if err != nil {
		conn.Close()
		rp.Put(rb)
		wp.Put(wb)
		tr.Del(trd)
		log.Error("key: %s handshake failed error(%v)", key, err)
		return
	}

	// 心跳超时器
	trd.Key = key
	tr.Set(trd, hb)

	// increase tcp stat
	server.Stat.IncrTcpOnline()

	// hanshake ok start dispatch goroutine
	go server.dispatchTCP(key, conn, wr, wp, wb, ch)
	for {
		if p, err = ch.CliProto.Set(); err != nil {
			break
		}
		if err = p.ReadTCP(rr); err != nil {
			break
		}

		if p.Operation == define.OP_LOGOUT {
			// only break loop
			break
		} else {
			tr.Set(trd, hb) // 收到包重置超时器

			// 交由Logic处理
			if err = server.operator.Operate(key, p); err != nil {
				log.Error("operate msg failed.key:%s, msg:%s", key, p)
				continue // 忽略错误
			}
		}
		ch.CliProto.SetAdv() // 使用下一个空间, 这个交由dispatcher发送
		ch.Signal()
	} // end for

	if err != nil && err != io.EOF {
		log.Error("key: %s server tcp failed error(%v)", key, err)
	}
	b.Del(key)
	tr.Del(trd)
	rp.Put(rb)
	conn.Close()
	ch.Close()
	if err = server.operator.Disconnect(key); err != nil {
		log.Error("key: %s operator do disconnect error(%v)", key, err)
	}
	if Debug {
		log.Debug("key: %s server tcp goroutine exit", key)
	}

	// decrease tcp stat
	server.Stat.DecrTcpOnline()
	return
}

// dispatch accepts connections on the listener and serves requests
// for each incoming connection.  dispatch blocks; the caller typically
// invokes it in a go statement.
func (server *Server) dispatchTCP(key string, conn *net.TCPConn, wr *bufio.Writer, wp *bytes.Pool, wb *bytes.Buffer, ch *Channel) {
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
			log.Debug("key:%s dispatch msg:%v", key, *p)
		}

		closeConn := false

		switch p {
		case proto.ProtoFinish:
			if Debug {
				log.Debug("key: %s wakeup exit dispatch goroutine", key)
			}
			finish = true
			goto failed
		case proto.ProtoReady: // 一个响应已准备好
			// fetch message from svrbox(client send)
			for {
				if p, err = ch.CliProto.Get(); err != nil {
					err = nil // must be empty error
					break
				}
				if p.Operation != define.OP_NONE {
					if err = p.WriteTCP(wr); err != nil {
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
			if err = p.WriteTCP(wr); err != nil {
				goto failed
			}
		}
		// only hungry flush response
		if err = wr.Flush(); err != nil {
			break
		}

		if closeConn {
			break
		}
	}
failed:
	if err != nil {
		log.Error("key: %s dispatch tcp error(%v)", key, err)
	}
	conn.Close()
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

// auth for myim handshake with client, use rsa & aes.
func (server *Server) authTCP(rr *bufio.Reader, wr *bufio.Writer, p *proto.Proto) (key string, heartbeat time.Duration, err error) {
	if err = p.ReadTCP(rr); err != nil {
		return
	}
	if p.Operation != define.OP_LOGIN {
		log.Warn("auth operation not valid: %d", p.Operation)
		err = ErrOperation
		return
	}
	if key, heartbeat, err = server.operator.Connect(p); err != nil {
		return
	}
	p.Body = nil
	p.Operation = define.OP_LOGIN_REPLY
	if err = p.WriteTCP(wr); err != nil {
		return
	}
	err = wr.Flush()
	return
}
