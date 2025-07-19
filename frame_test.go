package framestream

import (
	"bufio"
	"bytes"
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

func TestFrameDataIsIndependentCopy(t *testing.T) {
	// Simulate a stream with two different frames
	buf := new(bytes.Buffer)
	// Frame 1: length 4, data 0x01 0x02 0x03 0x04
	buf.Write([]byte{0, 0, 0, 4, 1, 2, 3, 4})
	// Frame 2: length 4, data 0xAA 0xBB 0xCC 0xDD
	buf.Write([]byte{0, 0, 0, 4, 0xAA, 0xBB, 0xCC, 0xDD})

	fs := &Fstrm{
		buf:    make([]byte, DATA_FRAME_LENGTH_MAX),
		reader: bufio.NewReader(buf),
	}

	f1, err := fs.RecvFrame(false)
	if err != nil {
		t.Fatalf("Error reading frame 1: %v", err)
	}
	f2, err := fs.RecvFrame(false)
	if err != nil {
		t.Fatalf("Error reading frame 2: %v", err)
	}

	// Modify the data of the first frame
	f1.data[0] = 0xFF

	if f2.data[0] == 0xFF {
		t.Error("Frame data are not independent (modification propagated)")
	}

	// Modify the data of the second frame
	f2.data[1] = 0xEE
	if f1.data[1] == 0xEE {
		t.Error("Frame data are not independent (modification propagated)")
	}
}
