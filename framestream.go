package framestream

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/segmentio/kafka-go/compress"
)

const DATA_FRAME_LENGTH_MAX = 65536

var ErrFrameTooLarge = errors.New("frame too large error")
var ErrReaderNotReady = errors.New("reader not ready")

var bufferCompressedPool = sync.Pool{
	New: func() interface{} {
		buf := new(bytes.Buffer)
		buf.Grow(DATA_FRAME_LENGTH_MAX)
		return buf
	},
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, DATA_FRAME_LENGTH_MAX)
		return &buf
	},
}

/* Framestream */
type Fstrm struct {
	buf         []byte
	reader      *bufio.Reader
	writer      *bufio.Writer
	conn        net.Conn
	readtimeout time.Duration
	ctype       []byte
	handshake   bool
}

func NewFstrm(reader *bufio.Reader, writer *bufio.Writer, conn net.Conn, readtimeout time.Duration, ctype []byte, handshake bool) *Fstrm {
	fs := &Fstrm{
		buf:         make([]byte, DATA_FRAME_LENGTH_MAX),
		reader:      reader,
		writer:      writer,
		ctype:       ctype,
		conn:        conn,
		readtimeout: readtimeout,
		handshake:   handshake,
	}

	return fs
}

func (fs Fstrm) SendFrame(frame *Frame) (err error) {

	r := bytes.NewReader(frame.data)

	if _, err = r.WriteTo(fs.writer); err == nil {
		err = fs.writer.Flush()
	}
	return err
}

func (fs Fstrm) RecvFrame(timeout bool) (*Frame, error) {
	// enable read timeaout
	if timeout && fs.readtimeout != 0 {
		fs.conn.SetReadDeadline(time.Now().Add(fs.readtimeout))
		defer fs.conn.SetDeadline(time.Time{})
	}

	if fs.reader == nil {
		return nil, ErrReaderNotReady
	}

	// read frame len (4 bytes)
	var lenBuf [4]byte
	if _, err := io.ReadFull(fs.reader, lenBuf[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(lenBuf[:])

	// checking data to read according to the size of the buffer
	if n > DATA_FRAME_LENGTH_MAX {
		fs.reader.Reset(bufio.NewReader(fs.conn))
		return nil, ErrFrameTooLarge
	}

	bufPtr := bufferPool.Get().(*[]byte)
	buf := *bufPtr
	defer bufferPool.Put(bufPtr)

	control := false
	offset := 0

	// it is a control frame, read the next 4 bytes to get control length
	if n == 0 {
		control = true
		if _, err := io.ReadFull(fs.reader, lenBuf[:]); err != nil {
			return nil, err
		}
		n = binary.BigEndian.Uint32(lenBuf[:])
		binary.BigEndian.PutUint32(buf[:4], n)
		offset = 4
	}

	// read  binary data and push it in the buffer
	if _, err := io.ReadFull(fs.reader, buf[offset:offset+int(n)]); err != nil {
		return nil, err
	}

	frame := &Frame{
		data:    buf[:offset+int(n)],
		control: control,
	}

	return frame, nil
}

func (fs Fstrm) SendCompressedFrame(codec compress.Codec, frame *Frame) (err error) {
	compressBuf := bufferCompressedPool.Get().(*bytes.Buffer)
	compressBuf.Reset()
	defer func() {
		compressBuf.Truncate(0)
		bufferCompressedPool.Put(compressBuf)
	}()

	compressor := codec.NewWriter(compressBuf)
	defer compressor.Close()

	_, err = compressor.Write(frame.data)
	if err != nil {
		return fmt.Errorf("compression failed: %w", err)
	}
	if err = compressor.Close(); err != nil {
		return fmt.Errorf("failed to close compressor: %w", err)
	}

	compressFrame := &Frame{}
	compressFrame.Write(compressBuf.Bytes())

	if err = fs.SendFrame(compressFrame); err != nil {
		return fmt.Errorf("failed to send compressed frame: %w", err)
	}
	return nil
}

func (fs Fstrm) RecvCompressedFrame(codec compress.Codec, timeout bool) (*Frame, error) {
	frame, err := fs.RecvFrame(timeout)
	if err != nil {
		return nil, err
	}

	compressReader := codec.NewReader(bytes.NewReader(frame.data))
	defer compressReader.Close()

	decompressedData, err := io.ReadAll(compressReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress frame: %w", err)
	}

	return &Frame{data: decompressedData}, nil
}

func (fs Fstrm) ProcessFrame(ch chan []byte) error {
	var err error
	var frame *Frame
	for {
		frame, err = fs.RecvFrame(false)
		if err != nil {
			break
		}
		if frame.control {
			if err = fs.ResetReceiver(frame); err != nil {
				break
			}
		}
		ch <- frame.data
	}
	return err
}

func (fs Fstrm) RecvControl() (*ControlFrame, error) {
	// waiting incoming frame
	frame, err := fs.RecvFrame(true)
	if err != nil {
		return nil, err
	}

	// checking if we have a control frame
	if !frame.control {
		return nil, ErrControlFrameExpected
	}

	// decode-it
	ctrl_frame := &ControlFrame{data: frame.data}
	if err := ctrl_frame.Decode(); err != nil {
		return nil, err
	}

	return ctrl_frame, nil
}

func (fs Fstrm) SendControl(control *ControlFrame) (err error) {
	if err := control.Encode(); err != nil {
		return err
	}
	frame := &Frame{control: true}
	if err := frame.Write(control.data); err != nil {
		return err
	}
	if err := fs.SendFrame(frame); err != nil {
		return err
	}
	return nil
}

func (fs Fstrm) InitSender() error {
	// handshake mode enabled
	if fs.handshake {
		// send ready control
		ctrl_ready := &ControlFrame{ctype: CONTROL_READY, ctypes: [][]byte{fs.ctype}}
		if err := fs.SendControl(ctrl_ready); err != nil {
			return err
		}

		// wait accept control
		ctrl, err := fs.RecvControl()
		if err != nil {
			return err
		}
		if ctrl.ctype != CONTROL_ACCEPT {
			return ErrControlFrameUnexpected
		}
		if !ctrl.CheckContentType(fs.ctype) {
			return ErrControlFrameContentTypeUnsupported
		}
	}

	// send start control frame
	ctrl_start := &ControlFrame{ctype: CONTROL_START, ctypes: [][]byte{fs.ctype}}
	if err := fs.SendControl(ctrl_start); err != nil {
		return err
	}

	return nil
}

func (fs Fstrm) ResetSender() error {
	// send stop control frame
	ctrl_stop := &ControlFrame{ctype: CONTROL_STOP}
	if err := fs.SendControl(ctrl_stop); err != nil {
		return err
	}

	// handshake mode enabled
	if fs.handshake {
		// wait finish control
		ctrl, err := fs.RecvControl()
		if err != nil {
			return err
		}
		if ctrl.ctype != CONTROL_FINISH {
			return ErrControlFrameUnexpected
		}
	}

	return nil
}

func (fs Fstrm) InitReceiver() error {
	// handshake?
	if fs.handshake {
		// wait ready control
		ctrl, err := fs.RecvControl()
		if err != nil {
			return err
		}
		if ctrl.ctype != CONTROL_READY {
			return ErrControlFrameUnexpected
		}
		if !ctrl.CheckContentType(fs.ctype) {
			return ErrControlFrameContentTypeUnsupported
		}

		// send accept control
		ctrl_accept := &ControlFrame{ctype: CONTROL_ACCEPT, ctypes: [][]byte{fs.ctype}}
		if err := fs.SendControl(ctrl_accept); err != nil {
			return err
		}
	}

	// decode start control frame
	ctrl, err := fs.RecvControl()
	if err != nil {
		return err
	}
	if ctrl.ctype != CONTROL_START {
		return ErrControlFrameUnexpected
	}
	if !ctrl.CheckContentType(fs.ctype) {
		return ErrControlFrameContentTypeUnsupported
	}

	return nil
}

func (fs Fstrm) ResetReceiver(frame *Frame) error {
	// decode stop control frame
	ctrl := ControlFrame{data: frame.data}
	if err := ctrl.Decode(); err != nil {
		return err
	}
	if ctrl.ctype != CONTROL_STOP {
		return ErrControlFrameUnexpected
	}

	// bidirectional mode
	if fs.handshake {
		// send finish control
		ctrl_finish := &ControlFrame{ctype: CONTROL_FINISH}
		if err := fs.SendControl(ctrl_finish); err != nil {
			return err
		}
	}

	return io.EOF
}
