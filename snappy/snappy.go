package snappy

import (
	"encoding/binary"
	"io"
	"sync"

	"github.com/golang/snappy"
	kafka "github.com/segmentio/kafka-go"
)

func init() {
	kafka.RegisterCompressionCodec(func() kafka.CompressionCodec {
		return NewCompressionCodec()
	})
}

type CompressionCodec struct{}

const Code = 2

func NewCompressionCodec() CompressionCodec {
	return CompressionCodec{}
}

// Code implements the kafka.CompressionCodec interface.
func (CompressionCodec) Code() int8 {
	return Code
}

// Encode implements the kafka.CompressionCodec interface.
func (CompressionCodec) Encode(src []byte) ([]byte, error) {
	// NOTE : passing a nil dst means snappy will allocate it.
	return snappy.Encode(nil, src), nil
}

// Decode implements the kafka.CompressionCodec interface.
func (CompressionCodec) Decode(src []byte) ([]byte, error) {
	return decode(src)
}

// NewReader implements the kafka.CompressionCodec interface.
func (CompressionCodec) NewReader(r io.Reader) io.ReadCloser {
	x := readerPool.Get().(*xerialReader)
	x.Reset(r)
	return &reader{x}
}

// NewWriter implements the kafka.CompressionCodec interface.
func (CompressionCodec) NewWriter(w io.Writer) io.WriteCloser {
	x := writerPool.Get().(*xerialWriter)
	x.Reset(w)
	return &writer{x}
}

type reader struct{ *xerialReader }

func (r *reader) Close() (err error) {
	if x := r.xerialReader; x != nil {
		r.xerialReader = nil
		x.Reset(nil)
		readerPool.Put(x)
	}
	return
}

type writer struct{ *xerialWriter }

func (w *writer) Close() (err error) {
	if x := w.xerialWriter; x != nil {
		w.xerialWriter = nil
		err = x.Flush()
		x.Reset(nil)
		writerPool.Put(x)
	}
	return
}

var readerPool = sync.Pool{
	New: func() interface{} {
		return &xerialReader{decode: snappy.Decode}
	},
}

var writerPool = sync.Pool{
	New: func() interface{} {
		return &xerialWriter{encode: snappy.Encode}
	},
}

// From github.com/eapache/go-xerial-snappy
func decode(src []byte) ([]byte, error) {
	if !isXerialHeader(src) {
		return snappy.Decode(nil, src)
	}

	var (
		pos   = uint32(16)
		max   = uint32(len(src))
		dst   = make([]byte, 0, len(src))
		chunk []byte
		err   error
	)

	for pos < max {
		size := binary.BigEndian.Uint32(src[pos : pos+4])
		pos += 4

		chunk, err = snappy.Decode(chunk, src[pos:pos+size])
		if err != nil {
			return nil, err
		}
		pos += size
		dst = append(dst, chunk...)
	}

	return dst, nil
}
