package framestream

import (
	"bytes"
	"encoding/binary"
)

/* Frame struct

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

func (frame Frame) Data() []byte {
	return frame.data
}

func (frame *Frame) Write(payload []byte) error {
	var buf bytes.Buffer
	var flen uint32

	// if it is a control frame then the frame length must be equal to zero
	if frame.control {
		flen = uint32(0)
	} else {
		flen = uint32(len(payload))
	}

	// add frame length
	if err := binary.Write(&buf, binary.BigEndian, flen); err != nil {
		return err
	}

	// append payload in the buffer
	if _, err := buf.Write(payload); err != nil {
		return err
	}

	frame.data = buf.Bytes()
	return nil
}
