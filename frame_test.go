package framestream

import "testing"

func TestFrameWrite(t *testing.T) {
	frame := &Frame{}

	ctrl_frame := &ControlFrame{ctype: CONTROL_ACCEPT, ctypes: [][]byte{[]byte("protobuf:dnstap.Dnstap")}}

	if err := ctrl_frame.Encode(); err != nil {
		t.Errorf("error to encode control frame %s", err)
	}

	if err := frame.Write(ctrl_frame.data); err != nil {
		t.Errorf("error to write frame %s", err)
	}
}

func BenchmarkFrameWrite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		frame := &Frame{}
		ctrl_frame := &ControlFrame{ctype: CONTROL_ACCEPT, ctypes: [][]byte{[]byte("protobuf:dnstap.Dnstap")}}

		if err := ctrl_frame.Encode(); err != nil {
			break
		}
		if err := frame.Write(ctrl_frame.data); err != nil {
			break
		}
	}
}
