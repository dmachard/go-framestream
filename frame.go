package framestream

import (
	"bytes"
	"encoding/binary"
)

/*
	Frame struct

|------------------------------------|----------------------|
| Data length                        | 4 bytes              |
|------------------------------------|----------------------|
| Payload                            | xx bytes             |
|------------------------------------|----------------------|

If the data length is equal to zero then it's a control frame
otherwise we have a data frame.
*/
type Frame struct {
	data    []byte
	control bool
}

func (frame Frame) Len() int {
	return len(frame.data)
}

func (frame Frame) IsControl() bool {
	return frame.control
}

func (frame Frame) Data() []byte {
	return frame.data
}

func (frame *Frame) Write(payload []byte) error {
	var flen uint32

	// if it is a control frame then the frame length must be equal to zero
	if frame.control {
		flen = uint32(0)
	} else {
		flen = uint32(len(payload))
	}

	// allocate exactly the size needed: 4 bytes for header + payload length
	frame.data = make([]byte, 4+len(payload))

	// add frame length
	binary.BigEndian.PutUint32(frame.data[:4], flen)

	// append payload in the buffer
	copy(frame.data[4:], payload)

	return nil
}

func (frame *Frame) AppendData(payload []byte) error {
	frame.data = append(frame.data, payload...)
	return nil
}

func (frame *Frame) Encode() error {
	var buf bytes.Buffer
	length := len(frame.data)
	if err := binary.Write(&buf, binary.BigEndian, uint32(length)); err != nil {
		return err
	}

	frame.data = append(buf.Bytes(), frame.data...)
	return nil
}
