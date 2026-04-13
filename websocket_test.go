package discord

import (
	"encoding/binary"
	"testing"
)

// ---------------------------------------------------------------------------
// maxFramePayload constant
// ---------------------------------------------------------------------------

func TestMaxFramePayload_SanityCheck(t *testing.T) {
	// 64 MiB should be well above any real Discord payload but safely below
	// typical process memory limits.
	const expected = 64 << 20
	if maxFramePayload != expected {
		t.Errorf("maxFramePayload = %d, want %d (64 MiB)", maxFramePayload, expected)
	}
}

// ---------------------------------------------------------------------------
// Frame length overflow guard
// ---------------------------------------------------------------------------

// buildFakeFrameHeader builds the first two bytes + extended-length bytes of a
// WebSocket frame header (no mask, text frame).
func buildFakeFrameHeader(length uint64) []byte {
	// byte 0: FIN=1, opcode=0x1 (text)
	// byte 1: MASK=0, length indicator
	var hdr []byte
	hdr = append(hdr, 0x81) // FIN | text
	if length < 126 {
		hdr = append(hdr, byte(length))
	} else if length <= 0xFFFF {
		hdr = append(hdr, 126)
		ext := make([]byte, 2)
		binary.BigEndian.PutUint16(ext, uint16(length))
		hdr = append(hdr, ext...)
	} else {
		hdr = append(hdr, 127)
		ext := make([]byte, 8)
		binary.BigEndian.PutUint64(ext, length)
		hdr = append(hdr, ext...)
	}
	return hdr
}

// TestFrameHeaderBuilding verifies that our helper builds headers correctly,
// which indirectly tests our understanding of the WebSocket frame format used
// in readFrame.
func TestFrameHeaderBuilding(t *testing.T) {
	// Small payload (< 126 bytes): 2-byte header.
	hdr := buildFakeFrameHeader(10)
	if len(hdr) != 2 {
		t.Errorf("small payload header length = %d, want 2", len(hdr))
	}
	if hdr[1] != 10 {
		t.Errorf("small payload length byte = %d, want 10", hdr[1])
	}

	// Medium payload (126–65535 bytes): 4-byte header.
	hdr = buildFakeFrameHeader(1000)
	if len(hdr) != 4 {
		t.Errorf("medium payload header length = %d, want 4", len(hdr))
	}
	if hdr[1] != 126 {
		t.Errorf("medium payload indicator = %d, want 126", hdr[1])
	}

	// Large payload (> 65535 bytes): 10-byte header.
	hdr = buildFakeFrameHeader(1 << 20)
	if len(hdr) != 10 {
		t.Errorf("large payload header length = %d, want 10", len(hdr))
	}
	if hdr[1] != 127 {
		t.Errorf("large payload indicator = %d, want 127", hdr[1])
	}
}

// ---------------------------------------------------------------------------
// wsComputeAccept
// ---------------------------------------------------------------------------

func TestWsComputeAccept(t *testing.T) {
	// RFC 6455 §1.3 test vector.
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	want := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	if got := wsComputeAccept(key); got != want {
		t.Errorf("wsComputeAccept(%q) = %q, want %q", key, got, want)
	}
}
