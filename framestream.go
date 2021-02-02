package framestream

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

const DATA_FRAME_LENGTH_MAX = 4096

var ErrFrameTooLarge = errors.New("frame too large error")

/* Framestream */
type Fstrm struct {
	buf       []byte
	reader    *bufio.Reader
	writer    *bufio.Writer
	ctype     []byte
	handshake bool
}

func NewFstrm(reader *bufio.Reader, writer *bufio.Writer, ctype []byte, handshake bool) *Fstrm {
	return &Fstrm{
		buf:       make([]byte, DATA_FRAME_LENGTH_MAX),
		reader:    reader,
		writer:    writer,
		ctype:     ctype,
		handshake: handshake,
	}
}

func (fs Fstrm) SendFrame(frame *Frame) (err error) {
	r := bytes.NewReader(frame.data)

	if _, err := r.WriteTo(fs.writer); err == nil {
		err = fs.writer.Flush()
	}

	return err
}

func (fs Fstrm) RecvFrame() (*Frame, error) {
	// flag control frame
	cf := false

	// read frame len (4 bytes)
	var n uint32
	if err := binary.Read(fs.reader, binary.BigEndian, &n); err != nil {
		return nil, err
	}

	// checking data to read according to the size of the buffer
	if n > uint32(len(fs.buf)) {
		return nil, ErrFrameTooLarge
	}

	// it is a control frame, read the next 4 bytes to get control length
	if n == 0 {
		cf = true
		if err := binary.Read(fs.reader, binary.BigEndian, &n); err != nil {
			return nil, err
		}
	}

	// read  binary data and push it in the buffer
	if _, err := io.ReadFull(fs.reader, fs.buf[0:n]); err != nil {
		return nil, err
	}

	frame := &Frame{
		data:    make([]byte, n),
		control: cf,
	}
	copy(frame.data, fs.buf[0:n])
	return frame, nil
}

func (fs Fstrm) ProcessFrame(ch chan []byte) (err error) {
	for {
		frame, err := fs.RecvFrame()
		if err != nil {
			break
		}
		if frame.control == true {
			if err := fs.ResetReceiver(frame); err != nil {
				break
			}
		}
		ch <- frame.data
	}
	//close(ch)
	return err
}

func (fs Fstrm) RecvControl() (*ControlFrame, error) {
	// waiting incoming frame
	frame, err := fs.RecvFrame()
	if err != nil {
		return nil, err
	}

	// checking if we have a control frame
	if frame.control == false {
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
	return nil
}

func (fs Fstrm) ResetSender() error {
	return nil
}

func (fs Fstrm) InitReceiver() error {
	// bidirectional mode
	if fs.handshake {
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
		ctrl_finish := &ControlFrame{ctype: CONTROL_FINISH, ctypes: [][]byte{fs.ctype}}
		if err := fs.SendControl(ctrl_finish); err != nil {
			return err
		}
	}

	return io.EOF
}
