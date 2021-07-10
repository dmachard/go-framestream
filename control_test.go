package framestream

import (
	"testing"
)

var ctrl_frame_accept = []byte{0, 0, 0, 34, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 22, 112, 114, 111, 116, 111,
	98, 117, 102, 58, 100, 110, 115, 116, 97, 112, 46, 68, 110, 115, 116, 97, 112}

func TestControlEncode(t *testing.T) {
	frame := &ControlFrame{ctype: CONTROL_ACCEPT, ctypes: [][]byte{[]byte("protobuf:dnstap.Dnstap")}}

	if err := frame.Encode(); err != nil {
		t.Errorf("error to encode control frame %s", err)
	}
}

func TestControlDecode(t *testing.T) {
	frame := &ControlFrame{data: ctrl_frame_accept}

	if err := frame.Decode(); err != nil {
		t.Errorf("error to decode control frame %s", err)
	}
}

func TestControlDecodeError(t *testing.T) {
	// invalid control frame length
	frame := &ControlFrame{data: []byte{0, 0, 127, 127}}
	if err := frame.Decode(); err == nil {
		t.Errorf("decode should be failed, invalid control frame length")
	}

	// unsupported control frame
	frame = &ControlFrame{data: []byte{0, 0, 0, 34, 0, 0, 0, 8}}
	if err := frame.Decode(); err == nil {
		t.Errorf("decode should be failed, unsupported control frame")
	}

}

func BenchmarkControlDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		frame := &ControlFrame{data: ctrl_frame_accept}
		if err := frame.Decode(); err != nil {
			break
		}
	}
}

func BenchmarkControlEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		frame := &ControlFrame{ctype: CONTROL_ACCEPT, ctypes: [][]byte{[]byte("protobuf:dnstap.Dnstap")}}
		if err := frame.Encode(); err != nil {
			break
		}
	}
}
