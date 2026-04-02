package discord

// websocket.go — a minimal, zero-dependency WebSocket client.
//
// Implements RFC 6455 (WebSocket) over a TLS or plain TCP connection.
// Client frames are always masked as required by the spec.

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
)

// WebSocket opcodes (RFC 6455 §11.8).
const (
	wsOpContinuation byte = 0x0
	wsOpText         byte = 0x1
	wsOpBinary       byte = 0x2
	wsOpClose        byte = 0x8
	wsOpPing         byte = 0x9
	wsOpPong         byte = 0xA
)

// wsConn is a single WebSocket connection.
type wsConn struct {
	conn   net.Conn
	br     *bufio.Reader
	mu     sync.Mutex // serialises writes
	closed bool
}

// wsDial opens a WebSocket connection to rawURL (ws:// or wss://).
func wsDial(rawURL string) (*wsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("wsDial: bad url: %w", err)
	}

	host := u.Host
	var conn net.Conn

	switch u.Scheme {
	case "wss":
		if !strings.Contains(host, ":") {
			host += ":443"
		}
		conn, err = tls.Dial("tcp", host, &tls.Config{ServerName: u.Hostname()})
	case "ws":
		if !strings.Contains(host, ":") {
			host += ":80"
		}
		conn, err = net.Dial("tcp", host)
	default:
		return nil, fmt.Errorf("wsDial: unsupported scheme %q", u.Scheme)
	}
	if err != nil {
		return nil, err
	}

	return wsHandshake(conn, u)
}

// wsHandshake performs the HTTP/1.1 Upgrade handshake.
func wsHandshake(conn net.Conn, u *url.URL) (*wsConn, error) {
	// Generate a random 16-byte key and base64-encode it.
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		conn.Close()
		return nil, err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)

	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	req := "GET " + path + " HTTP/1.1\r\n" +
		"Host: " + u.Host + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"\r\n"

	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, err
	}

	br := bufio.NewReaderSize(conn, 65536)

	// Read the status line.
	statusLine, err := br.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, err
	}
	if !strings.Contains(statusLine, "101") {
		conn.Close()
		return nil, fmt.Errorf("websocket upgrade failed: %s", strings.TrimSpace(statusLine))
	}

	// Read and validate headers.
	expected := wsComputeAccept(key)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break // end of headers
		}
		if strings.HasPrefix(strings.ToLower(line), "sec-websocket-accept:") {
			parts := strings.SplitN(line, ":", 2)
			got := strings.TrimSpace(parts[1])
			if got != expected {
				conn.Close()
				return nil, fmt.Errorf("websocket: invalid Sec-WebSocket-Accept")
			}
		}
	}

	return &wsConn{conn: conn, br: br}, nil
}

// wsComputeAccept computes the expected Sec-WebSocket-Accept header value.
func wsComputeAccept(key string) string {
	const magic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magic))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// ---------------------------------------------------------------------------
// Reading
// ---------------------------------------------------------------------------

// readFrame reads a single WebSocket frame and returns (fin, opcode, payload).
func (ws *wsConn) readFrame() (fin bool, opcode byte, payload []byte, err error) {
	hdr := make([]byte, 2)
	if _, err = io.ReadFull(ws.br, hdr); err != nil {
		return
	}

	fin = hdr[0]&0x80 != 0
	opcode = hdr[0] & 0x0F
	masked := hdr[1]&0x80 != 0
	plen := int64(hdr[1] & 0x7F)

	switch plen {
	case 126:
		ext := make([]byte, 2)
		if _, err = io.ReadFull(ws.br, ext); err != nil {
			return
		}
		plen = int64(binary.BigEndian.Uint16(ext))
	case 127:
		ext := make([]byte, 8)
		if _, err = io.ReadFull(ws.br, ext); err != nil {
			return
		}
		plen = int64(binary.BigEndian.Uint64(ext))
	}

	var maskKey [4]byte
	if masked {
		if _, err = io.ReadFull(ws.br, maskKey[:]); err != nil {
			return
		}
	}

	payload = make([]byte, plen)
	if _, err = io.ReadFull(ws.br, payload); err != nil {
		return
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return
}

// ReadMessage reads a complete (possibly fragmented) WebSocket message.
// Control frames (ping/pong/close) are handled transparently.
func (ws *wsConn) ReadMessage() ([]byte, error) {
	var buf []byte
	for {
		fin, opcode, payload, err := ws.readFrame()
		if err != nil {
			return nil, err
		}

		switch opcode {
		case wsOpClose:
			// Respond with a close frame then signal EOF.
			_ = ws.writeFrameLocked(wsOpClose, nil)
			return nil, io.EOF

		case wsOpPing:
			// Echo the ping payload back as a pong.
			if err := ws.writeFrame(wsOpPong, payload); err != nil {
				return nil, err
			}
			continue

		case wsOpPong:
			continue

		case wsOpText, wsOpBinary:
			buf = payload

		case wsOpContinuation:
			buf = append(buf, payload...)
		}

		if fin {
			return buf, nil
		}
	}
}

// ---------------------------------------------------------------------------
// Writing
// ---------------------------------------------------------------------------

// writeFrame sends a single WebSocket frame (thread-safe).
func (ws *wsConn) writeFrame(opcode byte, payload []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	return ws.writeFrameLocked(opcode, payload)
}

// writeFrameLocked sends a frame; caller must hold ws.mu.
func (ws *wsConn) writeFrameLocked(opcode byte, payload []byte) error {
	// Clients MUST mask all frames (RFC 6455 §5.3).
	var maskKey [4]byte
	if _, err := rand.Read(maskKey[:]); err != nil {
		return err
	}

	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ maskKey[i%4]
	}

	// Build frame header: FIN bit + opcode, MASK bit + payload length.
	frame := []byte{0x80 | opcode}
	l := len(payload)
	switch {
	case l < 126:
		frame = append(frame, byte(l)|0x80)
	case l < 65536:
		frame = append(frame, 126|0x80, byte(l>>8), byte(l))
	default:
		frame = append(frame, 127|0x80,
			0, 0, 0, 0,
			byte(l>>24), byte(l>>16), byte(l>>8), byte(l))
	}
	frame = append(frame, maskKey[:]...)
	frame = append(frame, masked...)

	_, err := ws.conn.Write(frame)
	return err
}

// WriteJSON serialises v as JSON and sends it as a text frame.
func (ws *wsConn) WriteJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return ws.writeFrame(wsOpText, data)
}

// Close sends a close frame and shuts the connection down.
func (ws *wsConn) Close() error {
	ws.mu.Lock()
	if ws.closed {
		ws.mu.Unlock()
		return nil
	}
	ws.closed = true
	_ = ws.writeFrameLocked(wsOpClose, nil)
	ws.mu.Unlock()
	return ws.conn.Close()
}
