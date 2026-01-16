package lock

import "testing"

func TestIsArrowMarkerSuffix(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		wantMatch  bool
		wantRemove int
	}{
		{
			name:       "empty slice",
			data:       []byte{},
			wantMatch:  false,
			wantRemove: 1,
		},
		{
			name:       "single byte",
			data:       []byte{0x61}, // 'a'
			wantMatch:  false,
			wantRemove: 1,
		},
		{
			name:       "ends with ArrowUpMarker",
			data:       []byte{'a', 'b', 0x00, 0x01},
			wantMatch:  true,
			wantRemove: 2,
		},
		{
			name:       "ends with ArrowDownMarker",
			data:       []byte{'a', 'b', 0x00, 0x02},
			wantMatch:  true,
			wantRemove: 2,
		},
		{
			name:       "ends with ArrowLeftMarker",
			data:       []byte{'a', 'b', 0x00, 0x03},
			wantMatch:  true,
			wantRemove: 2,
		},
		{
			name:       "ends with ArrowRightMarker",
			data:       []byte{'a', 'b', 0x00, 0x04},
			wantMatch:  true,
			wantRemove: 2,
		},
		{
			name:       "does not end with marker",
			data:       []byte{'a', 'b', 'c'},
			wantMatch:  false,
			wantRemove: 1,
		},
		{
			name:       "marker in middle but not at end",
			data:       []byte{0x00, 0x01, 'x'},
			wantMatch:  false,
			wantRemove: 1,
		},
		{
			name:       "just the marker",
			data:       []byte{0x00, 0x01},
			wantMatch:  true,
			wantRemove: 2,
		},
		{
			name:       "null byte but wrong second byte",
			data:       []byte{'a', 0x00, 0x05},
			wantMatch:  false,
			wantRemove: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatch, gotRemove := IsArrowMarkerSuffix(tt.data)
			if gotMatch != tt.wantMatch {
				t.Errorf("IsArrowMarkerSuffix() match = %v, want %v", gotMatch, tt.wantMatch)
			}
			if gotRemove != tt.wantRemove {
				t.Errorf("IsArrowMarkerSuffix() remove = %v, want %v", gotRemove, tt.wantRemove)
			}
		})
	}
}

func TestSecureBuffer_Backspace(t *testing.T) {
	t.Run("backspace on empty buffer", func(t *testing.T) {
		sb := NewSecureBuffer()
		if sb.Backspace() {
			t.Error("Backspace() on empty buffer should return false")
		}
		if sb.Len() != 0 {
			t.Errorf("Len() = %d, want 0", sb.Len())
		}
	})

	t.Run("backspace after regular char", func(t *testing.T) {
		sb := NewSecureBuffer()
		sb.AppendRune('a')
		sb.AppendRune('b')

		if !sb.Backspace() {
			t.Error("Backspace() should return true")
		}
		if sb.Len() != 1 {
			t.Errorf("Len() = %d, want 1", sb.Len())
		}
		if string(sb.Bytes()) != "a" {
			t.Errorf("Bytes() = %q, want %q", sb.Bytes(), "a")
		}
	})

	t.Run("backspace after arrow marker", func(t *testing.T) {
		sb := NewSecureBuffer()
		sb.AppendRune('x')
		sb.AppendString(ArrowUpMarker)

		if !sb.Backspace() {
			t.Error("Backspace() should return true")
		}
		// Should remove 2 bytes (the arrow marker)
		if sb.Len() != 1 {
			t.Errorf("Len() = %d, want 1", sb.Len())
		}
		if string(sb.Bytes()) != "x" {
			t.Errorf("Bytes() = %q, want %q", sb.Bytes(), "x")
		}
	})

	t.Run("mixed sequence backspace", func(t *testing.T) {
		sb := NewSecureBuffer()
		sb.AppendRune('a')               // 1 byte
		sb.AppendString(ArrowUpMarker)   // 2 bytes
		sb.AppendRune('b')               // 1 byte

		// Backspace removes 'b' (1 byte)
		if !sb.Backspace() {
			t.Error("Backspace() should return true")
		}
		if sb.Len() != 3 { // "a" + ArrowUpMarker
			t.Errorf("after 1st backspace: Len() = %d, want 3", sb.Len())
		}

		// Backspace removes ArrowUpMarker (2 bytes)
		if !sb.Backspace() {
			t.Error("Backspace() should return true")
		}
		if sb.Len() != 1 { // "a"
			t.Errorf("after 2nd backspace: Len() = %d, want 1", sb.Len())
		}

		// Backspace removes 'a' (1 byte)
		if !sb.Backspace() {
			t.Error("Backspace() should return true")
		}
		if sb.Len() != 0 {
			t.Errorf("after 3rd backspace: Len() = %d, want 0", sb.Len())
		}

		// Backspace on empty
		if sb.Backspace() {
			t.Error("Backspace() on empty should return false")
		}
	})

	t.Run("backspace with multiple arrow markers", func(t *testing.T) {
		sb := NewSecureBuffer()
		sb.AppendString(ArrowDownMarker)  // 2 bytes
		sb.AppendString(ArrowLeftMarker)  // 2 bytes
		sb.AppendString(ArrowRightMarker) // 2 bytes

		// Remove ArrowRightMarker
		sb.Backspace()
		if sb.Len() != 4 {
			t.Errorf("after 1st backspace: Len() = %d, want 4", sb.Len())
		}

		// Remove ArrowLeftMarker
		sb.Backspace()
		if sb.Len() != 2 {
			t.Errorf("after 2nd backspace: Len() = %d, want 2", sb.Len())
		}

		// Remove ArrowDownMarker
		sb.Backspace()
		if sb.Len() != 0 {
			t.Errorf("after 3rd backspace: Len() = %d, want 0", sb.Len())
		}
	})
}
