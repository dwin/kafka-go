package lz4

import (
	"io"
	"sync"

	"github.com/pierrec/lz4"
	kafka "github.com/segmentio/kafka-go"
)

var (
	readerPool = sync.Pool{
		New: func() interface{} { return lz4.NewReader(nil) },
	}

	writerPool = sync.Pool{
		New: func() interface{} { return lz4.NewWriter(nil) },
	}
)

func init() {
	kafka.RegisterCompressionCodec(func() kafka.CompressionCodec {
		return NewCompressionCodec()
	})
}

type CompressionCodec struct{}

const Code = 3

func NewCompressionCodec() CompressionCodec {
	return CompressionCodec{}
}

// Code implements the kafka.CompressionCodec interface.
func (CompressionCodec) Code() int8 {
	return Code
}

// NewReader implements the kafka.CompressionCodec interface.
func (CompressionCodec) NewReader(r io.Reader) io.ReadCloser {
	z := readerPool.Get().(*lz4.Reader)
	z.Reset(r)
	return &reader{z}
}

// NewWriter implements the kafka.CompressionCodec interface.
func (CompressionCodec) NewWriter(w io.Writer) io.WriteCloser {
	z := writerPool.Get().(*lz4.Writer)
	z.Reset(w)
	return &writer{z}
}

type reader struct{ *lz4.Reader }

func (r *reader) Close() (err error) {
	if z := r.Reader; z != nil {
		r.Reader = nil
		z.Reset(nil)
		readerPool.Put(z)
	}
	return
}

type writer struct{ *lz4.Writer }

func (w *writer) Close() (err error) {
	if z := w.Writer; z != nil {
		w.Writer = nil
		err = z.Close()
		z.Reset(nil)
		writerPool.Put(z)
	}
	return
}
