<img src="https://img.shields.io/badge/go%20version-min%201.21-green" alt="Go version"/>

# go-framestream

Frame Streams implementation in Go with compression support (gzip, lz4, snappy and zstd).

## Installation

```go
go get -u github.com/dmachard/go-framestream
```

## Usage example

Example to use the framestream library with Net.Pipe

```go
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
```

## Usage example with compression

```go
if err := fs_server.SendCompressedFrame(&compress.GzipCodec, frame); err != nil {
    t.Errorf("error to send frame: %s", err)
}
...
// receive frame, timeout 5s
frame, err := fs_client.RecvCompressedFrame(&compress.GzipCodec, true)
if err != nil {
    t.Errorf("error to receive frame: %s", err)
}
```

## Testing

```bash
$ go test -v
=== RUN   TestControlEncode
--- PASS: TestControlEncode (0.00s)
=== RUN   TestControlDecode
--- PASS: TestControlDecode (0.00s)
=== RUN   TestControlDecodeError
--- PASS: TestControlDecodeError (0.00s)
=== RUN   TestFrameWrite
--- PASS: TestFrameWrite (0.00s)
=== RUN   TestFramestreamHandshake
--- PASS: TestFramestreamHandshake (0.00s)
=== RUN   TestFramestreamData
--- PASS: TestFramestreamData (0.00s)
PASS
ok      github.com/dmachard/go-framestream
```
