package framestream

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const CONTROL_ACCEPT = 0x01
const CONTROL_START = 0x02
const CONTROL_STOP = 0x03
const CONTROL_READY = 0x04
const CONTROL_FINISH = 0x05

const CONTROL_FIELD_CONTENT_TYPE = 0x01

var ErrControlFrameMalformed = errors.New("control frame malformed")
var ErrControlFrameExpected = errors.New("control frame expected")
var ErrControlFrameUnexpected = errors.New("control frame unexpected")
var ErrControlFrameContentTypeUnsupported = errors.New("control frame with unsupported content type")

/* Control Frame struct
|------------------------------------|----------------------|
| Control frame length               | 4 bytes              |
|------------------------------------|----------------------|
| Control frame type                 | 4 bytes              |
|------------------------------------|----------------------|
| Control frame content type         | 4 bytes (optional)   |
|------------------------------------|----------------------|
| Control frame content type length  | 4 bytes (optional)   |
|------------------------------------|----------------------|
| Content type payload               | xx bytes             |
|------------------------------------|----------------------|
*/
type ControlFrame struct {
	data   []byte
	ctype  uint32
	ctypes [][]byte
}

func (ctrl *ControlFrame) Decode() error {
	// decoding content type
	ctrl.ctype = binary.BigEndian.Uint32(ctrl.data[:4])

	// decoding optional fields
	if len(ctrl.data[4:]) > 0 {
		cfields := ctrl.data[4:]
		for len(cfields) > 8 {
			cf_ctype := binary.BigEndian.Uint32(cfields[:4])
			if cf_ctype != CONTROL_FIELD_CONTENT_TYPE {
				return ErrControlFrameMalformed
			}
			cf_clen := binary.BigEndian.Uint32(cfields[4:8])
			ctrl.ctypes = append(ctrl.ctypes, cfields[8:cf_clen+8])
			cfields = cfields[cf_clen+8:]
		}

		if len(cfields) > 0 {
			return ErrControlFrameMalformed
		}
	}

	return nil
}

func (ctrl *ControlFrame) Encode() error {
	var buf bytes.Buffer

	// compute the control frame length
	cflen := 4 + len(ctrl.ctypes)*8
	for _, ctype := range ctrl.ctypes {
		cflen += len(ctype)
	}

	// add the control frame length
	if err := binary.Write(&buf, binary.BigEndian, uint32(cflen)); err != nil {
		return err
	}

	// add the control type
	if err := binary.Write(&buf, binary.BigEndian, uint32(ctrl.ctype)); err != nil {
		return err
	}

	// add optional fields
	for _, ctype := range ctrl.ctypes {
		// content type
		if err := binary.Write(&buf, binary.BigEndian, uint32(CONTROL_FIELD_CONTENT_TYPE)); err != nil {
			return err
		}

		// content type length
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(ctype))); err != nil {
			return err
		}

		// content type payload
		if _, err := buf.Write(ctype); err != nil {
			return err
		}
	}

	ctrl.data = buf.Bytes()
	return nil
}

func (ctrl *ControlFrame) CheckContentType(ctype []byte) bool {
	for _, cf_ctype := range ctrl.ctypes {
		if bytes.Compare(ctype, cf_ctype) == 0 {
			return true
		}
	}
	return false
}
