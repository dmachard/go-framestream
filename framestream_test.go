package framestream

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/segmentio/kafka-go/compress"
)

func TestFramestream_Handshake(t *testing.T) {
	client, server := net.Pipe()
	handshake := true

	// init framestream sender
	go func() {
		fs_server := NewFstrm(bufio.NewReader(server), bufio.NewWriter(server), server, 5*time.Second, []byte("frstrm"), handshake)
		if err := fs_server.InitSender(); err != nil {
			t.Errorf("error to init framestream sender: %s", err)
		}
	}()

	// init framestream receiver
	fs_client := NewFstrm(bufio.NewReader(client), bufio.NewWriter(client), client, 5*time.Second, []byte("frstrm"), handshake)
	if err := fs_client.InitReceiver(); err != nil {
		t.Errorf("error to init framestream receiver: %s", err)
	}
}

func TestRecvFrame_DataFrame(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		defer server.Close()
		// Frame data = [1,2,3,4]
		data := []byte{1, 2, 3, 4}
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, uint32(len(data)))
		buf.Write(data)
		server.Write(buf.Bytes())
	}()

	fs := NewFstrm(bufio.NewReader(client), bufio.NewWriter(client), client, 2*time.Second, []byte("ctype"), false)

	frame, err := fs.RecvFrame(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if frame.control {
		t.Fatalf("expected data frame, got control frame")
	}
	if !bytes.Equal(frame.Data(), []byte{1, 2, 3, 4}) {
		t.Fatalf("unexpected data: got %v", frame.Data())
	}
}

func TestRecvFrame_ReaderNotReady(t *testing.T) {
	fs := NewFstrm(nil, nil, nil, 0, []byte("ctype"), false)

	_, err := fs.RecvFrame(false)
	if !errors.Is(err, ErrReaderNotReady) {
		t.Fatalf("expected ErrReaderNotReady, got: %v", err)
	}
}

func TestRecvFrame_FrameTooLarge(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		defer server.Close()
		binary.Write(server, binary.BigEndian, uint32(DATA_FRAME_LENGTH_MAX+1))
	}()

	fs := NewFstrm(bufio.NewReader(client), bufio.NewWriter(client), client, 1*time.Second, []byte("ctype"), false)

	_, err := fs.RecvFrame(true)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge, got: %v", err)
	}
}

func TestFramestream_CompressedData(t *testing.T) {
	client, server := net.Pipe()
	handshake := true

	contentType := "protobuf:dnstap.Dnstap"
	frameData := []byte{1, 2, 3, 4}
	// init framestream sender
	go func() {
		fs_server := NewFstrm(bufio.NewReader(server), bufio.NewWriter(server), server, 5*time.Second, []byte(contentType), handshake)
		if err := fs_server.InitSender(); err != nil {
			t.Errorf("error to init framestream sender: %s", err)
		}

		// send frame
		frame := &Frame{}
		if err := frame.Write(frameData); err != nil {
			t.Errorf("error to init frame: %s", err)
		}
		if err := fs_server.SendCompressedFrame(&compress.GzipCodec, frame); err != nil {
			t.Errorf("error to send frame: %s", err)
		}
	}()

	// init framestream receiver
	fs_client := NewFstrm(bufio.NewReader(client), bufio.NewWriter(client), client, 5*time.Second, []byte(contentType), handshake)
	if err := fs_client.InitReceiver(); err != nil {
		t.Errorf("error to init framestream receiver: %s", err)
	}

	// receive frame, timeout 5s
	frame, err := fs_client.RecvCompressedFrame(&compress.GzipCodec, true)
	if err != nil {
		t.Errorf("error to receive frame: %s", err)
	}

	// read frame len (4 bytes)
	var n uint32
	if len(frame.Data()) < 4 {
		t.Errorf("error to read frame too short")
	}

	buf := bytes.NewReader(frame.Data())
	if err := binary.Read(buf, binary.BigEndian, &n); err != nil {
		t.Errorf("error to read frame len: %s", err)
	}

	if n > uint32(len(frame.Data()[4:])) {
		t.Errorf("no enough data")
	}

	data := frame.Data()[4 : 4+n]
	if !bytes.Equal(data, frameData) {
		t.Errorf("frame data not equal")
	}
}

func TestFramestream_SliceBoundsPanic_Issue974(t *testing.T) {
	client, server := net.Pipe()
	handshake := false // handshake not required for this test

	// Simulate a server that sends an oversized control frame
	go func() {
		defer server.Close()

		var payloadLen uint32 = 227195

		// Build a control frame:
		// - First 4 bytes: zero => indicates a control frame
		// - Next 4 bytes: actual length of the control payload
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, uint32(0))  // control frame indicator
		binary.Write(&buf, binary.BigEndian, payloadLen) // control payload length
		buf.Write(make([]byte, payloadLen))              // dummy payload (zeros)

		server.Write(buf.Bytes()) // send it to the client
	}()

	// Framestream client that will read the malformed frame
	fsClient := NewFstrm(bufio.NewReader(client), bufio.NewWriter(client), client, 2*time.Second, []byte("dummy"), handshake)

	_, err := fsClient.RecvFrame(true)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("unexpected error: got %v, want ErrFrameTooLarge", err)
	}
}
