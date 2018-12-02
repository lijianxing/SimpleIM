package proto

import (
	"encoding/json"
	"errors"
	"fmt"
	"myim/libs/bufio"
	"myim/libs/bytes"
	"myim/libs/define"
	"myim/libs/encoding/binary"
	"myim/libs/net/websocket"
)

// for tcp
const (
	MaxBodySize = int32(1 << 10)
)

const (
	// size
	PackSize      = 4
	HeaderSize    = 2
	VerSize       = 2
	OperationSize = 4
	SeqIdSize     = 4
	RawHeaderSize = PackSize + HeaderSize + VerSize + OperationSize + SeqIdSize
	MaxPackSize   = MaxBodySize + int32(RawHeaderSize)
	// offset
	PackOffset      = 0
	HeaderOffset    = PackOffset + PackSize
	VerOffset       = HeaderOffset + HeaderSize
	OperationOffset = VerOffset + VerSize
	SeqIdOffset     = OperationOffset + OperationSize
)

var (
	emptyProto    = Proto{}
	emptyJSONBody = []byte("{}")

	ErrProtoPackLen   = errors.New("default server codec pack length error")
	ErrProtoHeaderLen = errors.New("default server codec header length error")
)

var (
	ProtoReady  = &Proto{Operation: define.OP_PROTO_READY}
	ProtoFinish = &Proto{Operation: define.OP_PROTO_FINISH}
)

// Proto is a request&response written before every goim connect.  It is used internally
// but documented here as an aid to debugging, such as when analyzing
// network traffic.
// tcp:
// binary codec
// websocket & http:
// raw codec, with http header stored ver, operation, seqid
type Proto struct {
	Ver       int16           `json:"ver"`  // protocol version
	Operation int32           `json:"op"`   // operation for request
	SeqId     int32           `json:"seq"`  // sequence number chosen by client
	Body      json.RawMessage `json:"body"` // binary body bytes(json.RawMessage is []byte)
}

func (p *Proto) Reset() {
	*p = emptyProto
}

func (p *Proto) String() string {
	return fmt.Sprintf("\n-------- proto --------\nver: %d\nop: %d\nseq: %d\nbody: %v\n-----------------------", p.Ver, p.Operation, p.SeqId, p.Body)
}

func (p *Proto) WriteTo(b *bytes.Writer) {
	var (
		packLen = RawHeaderSize + int32(len(p.Body))
		buf     = b.Peek(RawHeaderSize)
	)
	binary.BigEndian.PutInt32(buf[PackOffset:], packLen)
	binary.BigEndian.PutInt16(buf[HeaderOffset:], int16(RawHeaderSize))
	binary.BigEndian.PutInt16(buf[VerOffset:], p.Ver)
	binary.BigEndian.PutInt32(buf[OperationOffset:], p.Operation)
	binary.BigEndian.PutInt32(buf[SeqIdOffset:], p.SeqId)
	if p.Body != nil {
		b.Write(p.Body)
	}
}

func (p *Proto) ReadTCP(rr *bufio.Reader) (err error) {
	var (
		bodyLen   int
		headerLen int16
		packLen   int32
		buf       []byte
	)
	if buf, err = rr.Pop(RawHeaderSize); err != nil {
		return
	}
	packLen = binary.BigEndian.Int32(buf[PackOffset:HeaderOffset])
	headerLen = binary.BigEndian.Int16(buf[HeaderOffset:VerOffset])
	p.Ver = binary.BigEndian.Int16(buf[VerOffset:OperationOffset])
	p.Operation = binary.BigEndian.Int32(buf[OperationOffset:SeqIdOffset])
	p.SeqId = binary.BigEndian.Int32(buf[SeqIdOffset:])
	if packLen > MaxPackSize {
		return ErrProtoPackLen
	}
	if headerLen != RawHeaderSize {
		return ErrProtoHeaderLen
	}
	if bodyLen = int(packLen - int32(headerLen)); bodyLen > 0 {
		p.Body, err = rr.Pop(bodyLen)
	} else {
		p.Body = nil
	}
	return
}

func (p *Proto) WriteTCP(wr *bufio.Writer) (err error) {
	var (
		buf     []byte
		packLen int32
	)

	packLen = RawHeaderSize + int32(len(p.Body))
	if buf, err = wr.Peek(RawHeaderSize); err != nil {
		return
	}
	binary.BigEndian.PutInt32(buf[PackOffset:], packLen)
	binary.BigEndian.PutInt16(buf[HeaderOffset:], int16(RawHeaderSize))
	binary.BigEndian.PutInt16(buf[VerOffset:], p.Ver)
	binary.BigEndian.PutInt32(buf[OperationOffset:], p.Operation)
	binary.BigEndian.PutInt32(buf[SeqIdOffset:], p.SeqId)
	if p.Body != nil {
		_, err = wr.Write(p.Body)
	}
	return
}

func (p *Proto) ReadWebsocket(ws *websocket.Conn) (err error) {
	op, data, err := ws.ReadMessage()
	if err != nil {
		return
	}

	if op == websocket.TextMessage {
		if err = json.Unmarshal(data, p); err != nil {
			return
		}
	} else {
		err = errors.New(fmt.Sprintf("not supported websocket op:%d", op))
	}
	return
}

func (p *Proto) WriteWebsocket(ws *websocket.Conn) (err error) {
	// json msg
	var msg []byte
	if msg, err = json.Marshal(p); err != nil {
		return
	}
	err = ws.WriteMessage(websocket.TextMessage, msg)
	return
}
