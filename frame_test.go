package framestream

import (
	"testing"
)

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

func TestFrameAppendData(t *testing.T) {
	frame := &Frame{}

	frameDataA := []byte{1, 2, 3, 4}
	frameDataB := []byte{1, 2}

	if err := frame.Write(frameDataA); err != nil {
		t.Errorf("error to append data frame %s", err)
	}

	if frame.Len() != 8 {
		t.Errorf("invalid data frame length")
	}

	if err := frame.AppendData(frameDataB); err != nil {
		t.Errorf("error to append data frame %s", err)
	}

	if frame.Len() != 10 {
		t.Errorf("invalid data frame length after second append")
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
