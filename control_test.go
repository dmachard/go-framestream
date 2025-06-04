package framestream

import (
	"testing"
)

var ctrl_frame_accept = []byte{0, 0, 0, 34, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 22, 112, 114, 111, 116, 111,
	98, 117, 102, 58, 100, 110, 115, 116, 97, 112, 46, 68, 110, 115, 116, 97, 112}

func TestControlEncode(t *testing.T) {
	frame := &ControlFrame{ctype: CONTROL_ACCEPT, ctypes: [][]byte{[]byte("protobuf:dnstap.Dnstap")}}

	if err := frame.Encode(); err != nil {
		t.Errorf("error to encode control frame %s", err)
	}
}

func TestControlDecode(t *testing.T) {
	frame := &ControlFrame{data: ctrl_frame_accept}

	if err := frame.Decode(); err != nil {
		t.Errorf("error to decode control frame %s", err)
	}
}

func TestControlDecodeError(t *testing.T) {
	// invalid control frame length
	frame := &ControlFrame{data: []byte{0, 0, 127, 127}}
	if err := frame.Decode(); err == nil {
		t.Errorf("decode should be failed, invalid control frame length")
	}

	// unsupported control frame
	frame = &ControlFrame{data: []byte{0, 0, 0, 34, 0, 0, 0, 8}}
	if err := frame.Decode(); err == nil {
		t.Errorf("decode should be failed, unsupported control frame")
	}

}

// related to the issue https://github.com/dmachard/DNS-collector/issues/1011
func TestControlDecodeWithProperErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		expectError bool
		errorType   error
	}{
		{
			name:        "empty_data",
			data:        []byte{},
			expectError: true,
			errorType:   ErrControlFrameMalformed,
		},
		{
			name:        "insufficient_data_4_bytes",
			data:        []byte{0, 0, 0, 4},
			expectError: true,
			errorType:   ErrControlFrameMalformed,
		},
		{
			name:        "insufficient_data_7_bytes",
			data:        []byte{0, 0, 0, 7, 0, 0, 0},
			expectError: true,
			errorType:   ErrControlFrameMalformed,
		},
		{
			name:        "valid_minimal_frame",
			data:        []byte{0, 0, 0, 4, 0, 0, 0, 1}, // 8 bytes total, cflen=4, CONTROL_ACCEPT
			expectError: false,
		},
		{
			name: "truncated_optional_fields",
			data: []byte{
				0, 0, 0, 12, // cflen = 12 (claiming there's optional data)
				0, 0, 0, 1, // ctype = CONTROL_ACCEPT
				0, 0, 0, 1, // cf_ctype = CONTROL_FIELD_CONTENT_TYPE (4 bytes)
				// Missing the next 4 bytes for cf_clen - will cause panic in loop
			},
			expectError: true,
			errorType:   ErrControlFrameMalformed,
		},
		{
			name: "truncated_optional_field_payload",
			data: []byte{
				0, 0, 0, 20, // cflen = 20
				0, 0, 0, 1, // ctype = CONTROL_ACCEPT
				0, 0, 0, 1, // cf_ctype = CONTROL_FIELD_CONTENT_TYPE
				0, 0, 0, 10, // cf_clen = 10 (claiming 10 bytes of payload)
				0, 0, // Only 2 bytes instead of 10 - will panic when accessing [8:18]
			},
			expectError: true,
			errorType:   ErrControlFrameMalformed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			frame := &ControlFrame{data: tc.data}
			err := frame.Decode()

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				if tc.errorType != nil && err != tc.errorType {
					t.Errorf("Expected error %v but got %v", tc.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func BenchmarkControlDecode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		frame := &ControlFrame{data: ctrl_frame_accept}
		if err := frame.Decode(); err != nil {
			break
		}
	}
}

func BenchmarkControlEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		frame := &ControlFrame{ctype: CONTROL_ACCEPT, ctypes: [][]byte{[]byte("protobuf:dnstap.Dnstap")}}
		if err := frame.Encode(); err != nil {
			break
		}
	}
}
