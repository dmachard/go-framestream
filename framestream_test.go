package framestream

import (
	"bufio"
	"bytes"
	"encoding/binary"
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

func TestFramestream_Data(t *testing.T) {
	client, server := net.Pipe()
	handshake := true

	// init framestream sender
	go func() {
		fs_server := NewFstrm(bufio.NewReader(server), bufio.NewWriter(server), server, 5*time.Second, []byte("frstrm"), handshake)
		if err := fs_server.InitSender(); err != nil {
			t.Errorf("error to init framestream sender: %s", err)
		}

		// send frame
		frame := &Frame{}
		if err := frame.Write([]byte{1, 2, 3, 4}); err != nil {
			t.Errorf("error to init frame: %s", err)
		}
		if err := fs_server.SendFrame(frame); err != nil {
			t.Errorf("error to send frame: %s", err)
		}
	}()

	// init framestream receiver
	fs_client := NewFstrm(bufio.NewReader(client), bufio.NewWriter(client), client, 5*time.Second, []byte("frstrm"), handshake)
	if err := fs_client.InitReceiver(); err != nil {
		t.Errorf("error to init framestream receiver: %s", err)
	}

	// receive frame, timeout 5s
	_, err := fs_client.RecvFrame(true)
	if err != nil {
		t.Errorf("error to receive frame: %s", err)
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
	if err == nil {
		t.Errorf("expected error due to oversized frame, but got none")
	} else {
		t.Logf("RecvFrame returned expected error: %v", err)
	}
}
