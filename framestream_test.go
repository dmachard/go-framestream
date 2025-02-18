package framestream

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/segmentio/kafka-go/compress"
)

// Mock connection pour tests
type mockConn struct {
	io.Reader
	io.Writer
}

func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func BenchmarkSendFrame(b *testing.B) {
	buf := new(bytes.Buffer)
	fs := NewFstrm(bufio.NewReader(buf), bufio.NewWriter(buf), nil, 0, nil, false)
	frame := &Frame{data: make([]byte, 1024)} // Frame de 1 KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fs.SendFrame(frame)
	}
}

func BenchmarkRecvFrame(b *testing.B) {
	buf := new(bytes.Buffer)
	conn := &mockConn{Reader: buf, Writer: buf}
	fs := NewFstrm(bufio.NewReader(buf), bufio.NewWriter(buf), conn, 0, nil, false)

	testFrame := &Frame{data: make([]byte, 1024)}
	fs.SendFrame(testFrame)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fs.RecvFrame(false)
	}
}

func BenchmarkSendCompressedFrame(b *testing.B) {
	buf := new(bytes.Buffer)
	fs := NewFstrm(bufio.NewReader(buf), bufio.NewWriter(buf), nil, 0, nil, false)
	frame := &Frame{data: make([]byte, 1024)} // Frame de 1 KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fs.SendCompressedFrame(&compress.SnappyCodec, frame)
	}
}

func BenchmarkRecvCompressedFrame(b *testing.B) {
	buf := new(bytes.Buffer)
	conn := &mockConn{Reader: buf, Writer: buf}
	fs := NewFstrm(bufio.NewReader(buf), bufio.NewWriter(buf), conn, 0, nil, false)

	testFrame := &Frame{data: make([]byte, 1024)}
	fs.SendCompressedFrame(&compress.SnappyCodec, testFrame)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fs.RecvCompressedFrame(&compress.SnappyCodec, false)
	}
}

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
