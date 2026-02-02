package framestream

import (
	"bufio"
	"bytes"
	"testing"
	"time"
)

// FuzzRecvFrame tests the RecvFrame method with random inputs
// to ensure it handles malformed data without panicking.
func FuzzRecvFrame(f *testing.F) {
	// Add some seed corpus
	f.Add([]byte{0, 0, 0, 4, 1, 2, 3, 4}) // Valid data frame
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 0}) // Valid control frame (empty)
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF}) // Huge length (should error)
	f.Add([]byte{0, 0, 0, 0})             // Incomplete control frame
	f.Add([]byte{0})                      // Tiny input

	f.Fuzz(func(t *testing.T, data []byte) {
		// Mock reader with fuzz data
		r := bytes.NewReader(data)
		bufReader := bufio.NewReader(r)

		// Use a mock/noop writer since we only test receiving
		fs := NewFstrm(bufReader, nil, nil, 0*time.Second, []byte("test"), false)

		// Call the function under test
		// We ignore the error because we expect errors on random data
		// We only care about panics or hangs (though hangs are hard to detect in fuzzing without timeouts)
		_, _ = fs.RecvFrame(false)
	})
}
