![Go version](https://img.shields.io/badge/go%20version-min%201.23-green)
![Go tests](https://img.shields.io/badge/go%20tests-24-green)
![Go bench](https://img.shields.io/badge/go%20microbench-11-green)
![Go fuzzing](https://img.shields.io/badge/go%20fuzzing-1-green)

# go-framestream

Frame Streams implementation in Go with channel-based support, **compression** (gzip, lz4, snappy and zstd), and **high-performance zero-copy reading**.

A fast alternative to [farsightsec/golang-framestream](https://github.com/farsightsec/golang-framestream).

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


## Frame size limits

By default, the library enforces the following limits:
- Control frames: 4064 bytes
- Data frames: 1048576 bytes

You can increase these limits to support vendors sending larger frames (like Infoblox):

```go
fs := NewFstrm(...)

// Increase control frame limit to 16KB
fs.SetControlFrameMaxLength(16384)

// Increase data frame limit to 1MB
fs.SetDataFrameMaxLength(1048576)
```

## High-Performance Zero-Copy Reader

For high-throughput environments (e.g. DNSTAP processing tens of thousands of frames per second), the library provides zero-copy reading modes that eliminate heap allocations on the critical receive path.

### Enabling Zero-Copy on `RecvFrame`

You can enable zero-copy on an existing `Fstrm` instance without changing any of your `RecvFrame` call sites:

```go
fs := NewFstrm(...)

// Enable zero-copy mode: RecvFrame will reuse an internal buffer (0 allocs/op)
fs.SetZeroCopy(true)

// Optional: pre-allocate internal buffer to avoid dynamic growth allocations
fs.InitViewBuffer(64 * 1024) // 64 KB

for {
    frame, err := fs.RecvFrame(false)
    if err != nil {
        break
    }
    // Process frame.Data() synchronously before the next read
    process(frame.Data())
}
```

> **Note:** In zero-copy mode, `frame.Data()` borrows memory from the reader's internal buffer and is overwritten on the next read. If your application processes frames asynchronously (e.g. sending raw slices over channels or storing them in queues), simply use the **default mode** (`SetZeroCopy(false)`), which automatically allocates independent, safe slices.

### Benchmark: `go-framestream` vs `farsightsec/golang-framestream`

Comparative benchmark decoding a stream of 50 data frames (512 bytes each):

| Metric | `go-framestream` | `farsightsec/golang-framestream` |
| :--- | :--- | :--- |
| **Zero-Copy / Fast Mode** | **`SetZeroCopy(true)`** | `Reader.ReadFrame()` |
| ↳ Speed | **961 ns/op** | 1 908 ns/op |
| ↳ Memory | **0 B/op** | 4 480 B/op |
| ↳ Allocations | **0 allocs/op** | 59 allocs/op |
| **Default Mode** | **`RecvFrame()`** | `Decoder.Decode()` |
| ↳ Speed | **5 056 ns/op** | 77 799 ns/op |
| ↳ Memory | **27 280 B/op** | 1 053 091 B/op |
| ↳ Allocations | 104 allocs/op | 61 allocs/op |

* **In Zero-Copy mode (`SetZeroCopy(true)`):**
  * **2x faster** than Farsight's low-level `Reader` with **zero heap allocations**.
  * **80x faster** than Farsight's default `Decoder`.
* **In Default mode (`RecvFrame()`):**
  * **15x faster** and allocates **38x less memory** than Farsight's default `Decoder` (which allocates 1 MB per decoder).
* In addition, `go-framestream` supports **compression** (gzip, zstd, lz4, snappy) and raw unhandshaked streams, neither of which are supported by `farsightsec`.

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
cpu: AMD Ryzen 9 9900X 12-Core Processor            
BenchmarkControlDecode-24               61187217        19.02 ns/op          24 B/op        1 allocs/op
BenchmarkControlEncode-24               77231541        14.68 ns/op          48 B/op        1 allocs/op
BenchmarkFrameWrite-24                  39990684        29.20 ns/op          96 B/op        2 allocs/op
BenchmarkFrameEncode-24                135106399         8.65 ns/op          16 B/op        1 allocs/op
BenchmarkRecvFrame_RawDataFrame-24       1580408       765.4 ns/op         3136 B/op       28 allocs/op
BenchmarkRecvFrame_ZeroCopyFlag-24       5590976       218.8 ns/op            0 B/op        0 allocs/op
PASS
ok      github.com/dmachard/go-framestream
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
