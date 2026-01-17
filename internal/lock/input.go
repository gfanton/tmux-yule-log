package lock

import (
	"unicode/utf8"

	"github.com/awnumar/memguard"
	"github.com/gdamore/tcell/v2"
)

// ---- Arrow Key Markers
// Internal markers for arrow keys in passphrase (won't appear in normal input).
// Uses NULL + control character sequences that can't be typed.

const (
	ArrowUpMarker    = "\x00\x01" // NULL + SOH
	ArrowDownMarker  = "\x00\x02" // NULL + STX
	ArrowLeftMarker  = "\x00\x03" // NULL + ETX
	ArrowRightMarker = "\x00\x04" // NULL + EOT
)

// ArrowKeyMarker returns the marker string for an arrow key.
func ArrowKeyMarker(key tcell.Key) string {
	switch key {
	case tcell.KeyUp:
		return ArrowUpMarker
	case tcell.KeyDown:
		return ArrowDownMarker
	case tcell.KeyLeft:
		return ArrowLeftMarker
	case tcell.KeyRight:
		return ArrowRightMarker
	default:
		return ""
	}
}

// ArrowKeyDisplay returns a display character for an arrow key.
func ArrowKeyDisplay(key tcell.Key) rune {
	switch key {
	case tcell.KeyUp:
		return '\u2191' // ↑
	case tcell.KeyDown:
		return '\u2193' // ↓
	case tcell.KeyLeft:
		return '\u2190' // ←
	case tcell.KeyRight:
		return '\u2192' // →
	default:
		return ' '
	}
}

// IsArrowKey checks if the given key is an arrow key.
func IsArrowKey(key tcell.Key) bool {
	return key == tcell.KeyUp || key == tcell.KeyDown ||
		key == tcell.KeyLeft || key == tcell.KeyRight
}

// IsArrowMarkerSuffix checks if data ends with an arrow marker.
// Returns true and the marker length (2) if found, false and 1 otherwise.
func IsArrowMarkerSuffix(data []byte) (bool, int) {
	if len(data) < 2 {
		return false, 1
	}
	last2 := string(data[len(data)-2:])
	switch last2 {
	case ArrowUpMarker, ArrowDownMarker, ArrowLeftMarker, ArrowRightMarker:
		return true, 2
	}
	return false, 1
}

// ClearBytes securely wipes a byte slice by overwriting with zeros.
func ClearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// ---- Secure Input Buffer

// SecureBuffer wraps memguard for secure password handling.
type SecureBuffer struct {
	enclave *memguard.Enclave
	data    []byte
}

// NewSecureBuffer creates a new secure buffer for password input.
func NewSecureBuffer() *SecureBuffer {
	return &SecureBuffer{
		data: make([]byte, 0, 256),
	}
}

// Append adds data to the secure buffer.
func (sb *SecureBuffer) Append(data []byte) {
	sb.data = append(sb.data, data...)
}

// AppendRune adds a rune to the secure buffer, properly encoding UTF-8.
func (sb *SecureBuffer) AppendRune(r rune) {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	sb.data = append(sb.data, buf[:n]...)
}

// AppendString adds a string to the secure buffer.
func (sb *SecureBuffer) AppendString(s string) {
	sb.data = append(sb.data, s...)
}

// Backspace removes the last character from the buffer.
// Handles multi-byte arrow markers correctly.
// Returns true if data was removed.
func (sb *SecureBuffer) Backspace() bool {
	if len(sb.data) == 0 {
		return false
	}

	isArrow, removeLen := IsArrowMarkerSuffix(sb.data)
	if isArrow {
		sb.data = sb.data[:len(sb.data)-removeLen]
	} else {
		sb.data = sb.data[:len(sb.data)-1]
	}
	return true
}

// Clear securely wipes and resets the buffer.
func (sb *SecureBuffer) Clear() {
	// Overwrite with zeros before truncating
	for i := range sb.data {
		sb.data[i] = 0
	}
	sb.data = sb.data[:0]
}

// Len returns the length of the buffer.
func (sb *SecureBuffer) Len() int {
	return len(sb.data)
}

// Bytes returns a copy of the buffer contents.
// The returned slice should be cleared after use.
func (sb *SecureBuffer) Bytes() []byte {
	result := make([]byte, len(sb.data))
	copy(result, sb.data)
	return result
}

// Seal moves the data into a memguard enclave for secure storage.
// After calling Seal, the buffer is cleared and cannot be used until Open.
func (sb *SecureBuffer) Seal() {
	if len(sb.data) == 0 {
		return
	}

	sb.enclave = memguard.NewEnclave(sb.data)
	sb.Clear()
}

// Open retrieves the data from the enclave.
// Returns a LockedBuffer that should be destroyed after use.
func (sb *SecureBuffer) Open() (*memguard.LockedBuffer, error) {
	if sb.enclave == nil {
		return memguard.NewBufferFromBytes(sb.data), nil
	}
	return sb.enclave.Open()
}

// Destroy securely wipes all data.
// The enclave is set to nil, allowing GC to collect it.
// memguard.Enclave doesn't expose a Destroy method; its internal
// LockedBuffer is wiped when the Enclave is garbage collected.
func (sb *SecureBuffer) Destroy() {
	sb.Clear()
	sb.enclave = nil
}

// VisualLen returns the number of visual characters (arrows count as 1).
func (sb *SecureBuffer) VisualLen() int {
	count := 0
	data := sb.data
	for len(data) > 0 {
		if len(data) >= 2 {
			marker := string(data[:2])
			if marker == ArrowUpMarker || marker == ArrowDownMarker ||
				marker == ArrowLeftMarker || marker == ArrowRightMarker {
				count++
				data = data[2:]
				continue
			}
		}
		count++
		data = data[1:]
	}
	return count
}
