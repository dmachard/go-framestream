<img src="https://img.shields.io/badge/go%20version-min%201.18-green" alt="Go version"/>

# go-framestream

Frame Streams implementation in Go with channed-base support and **compression** (gzip, lz4, snappy and zstd).

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


```bash
$ go test -bench=. -benchmem
goos: linux
goarch: amd64
pkg: github.com/dmachard/go-framestream
cpu: Intel(R) Core(TM) i5-7200U CPU @ 2.50GHz
BenchmarkControlDecode-4                19755384                59.40 ns/op           24 B/op          1 allocs/op
BenchmarkControlEncode-4                 4363948               232.9 ns/op           128 B/op          6 allocs/op
BenchmarkFrameWrite-4                    3096256               424.3 ns/op           244 B/op          9 allocs/op
BenchmarkRecvFrame_RawDataFrame-4         635953              1944 ns/op             508 B/op         29 allocs/op
PASS
ok      github.com/dmachard/go-framestream      10.135s
```

### Frame format

Data Frame

| Headers                            | Bytes              |
|------------------------------------|--------------------|
| Data length                        | 4 bytes            |
| Payload                            | xx bytes           |

Control Frame

| Headers                            | Bytes              |
|------------------------------------|----------------------|
| Control frame length               | 4 bytes              |
| Control frame type                 | 4 bytes              |
| Control frame content type         | 4 bytes (optional)   |
| Control frame content type length  | 4 bytes (optional)   |
| Content type payload               | xx bytes             |
