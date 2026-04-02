package discord

// gateway.go — Discord Gateway v10 client.
//
// Responsibilities:
//   - Establish a TLS WebSocket connection to the Discord Gateway.
//   - Send the Identify payload to authenticate.
//   - Maintain the heartbeat loop required by Discord.
//   - Dispatch incoming events to the event dispatcher.
//   - Automatically reconnect and resume a session after disconnects.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"
)

// Discord Gateway opcodes (https://discord.com/developers/docs/topics/opcodes-and-status-codes).
const (
	opDispatch            = 0
	opHeartbeat           = 1
	opIdentify            = 2
	opPresenceUpdate      = 3
	opResume              = 6
	opReconnect           = 7
	opInvalidSession      = 9
	opHello               = 10
	opHeartbeatACK        = 11
)

const (
	gatewayURL    = "wss://gateway.discord.gg/?v=10&encoding=json"
	gatewayAPIVer = 10
)

// gateway manages the Discord Gateway connection for a Bot.
type gateway struct {
	bot *Bot

	// Connection state.
	conn      *wsConn
	sequence  int64 // last seen sequence number (atomic)
	sessionID string
	resumeURL string

	// Heartbeat state.
	heartbeatInterval time.Duration
	lastACK           atomic.Bool

	// Lifecycle.
	stopCh chan struct{}
	doneCh chan struct{}
}

func newGateway(bot *Bot) *gateway {
	return &gateway{
		bot:    bot,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// start begins the connection loop in the background and returns immediately.
func (g *gateway) start() error {
	// Perform the first connection synchronously so the caller gets an
	// immediate error if the token or network is bad.
	if err := g.connect(); err != nil {
		return err
	}
	go g.loop()
	return nil
}

// stop signals the gateway to shut down cleanly.
func (g *gateway) stop() {
	close(g.stopCh)
	if g.conn != nil {
		g.conn.Close()
	}
	<-g.doneCh
}

// loop drives reconnection. It runs until stop() is called.
func (g *gateway) loop() {
	defer close(g.doneCh)

	for {
		// readLoop blocks until the connection drops.
		err := g.readLoop()

		// Check whether stop() was called.
		select {
		case <-g.stopCh:
			return
		default:
		}

		if err != nil && err != io.EOF {
			log.Printf("[gateway] disconnected: %v — reconnecting in 5s", err)
		} else {
			log.Printf("[gateway] disconnected — reconnecting in 5s")
		}

		// Back off before reconnecting.
		select {
		case <-time.After(5 * time.Second):
		case <-g.stopCh:
			return
		}

		if err := g.connect(); err != nil {
			log.Printf("[gateway] reconnect failed: %v", err)
		}
	}
}

// connect opens a new WebSocket connection and sends Identify (or Resume).
func (g *gateway) connect() error {
	target := gatewayURL
	if g.resumeURL != "" {
		target = g.resumeURL + "?v=10&encoding=json"
	}

	conn, err := wsDial(target)
	if err != nil {
		return fmt.Errorf("gateway: dial: %w", err)
	}
	g.conn = conn
	return nil
}

// ---------------------------------------------------------------------------
// Read loop
// ---------------------------------------------------------------------------

// readLoop reads frames from the WebSocket until an error occurs.
// It handles all opcodes and dispatches events.
func (g *gateway) readLoop() error {
	for {
		data, err := g.conn.ReadMessage()
		if err != nil {
			return err
		}

		var p gatewayPayload
		if err := json.Unmarshal(data, &p); err != nil {
			log.Printf("[gateway] unmarshal error: %v", err)
			continue
		}

		if err := g.handlePayload(&p); err != nil {
			return err
		}
	}
}

// handlePayload processes a single gateway payload.
func (g *gateway) handlePayload(p *gatewayPayload) error {
	// Update sequence number for every dispatch event.
	if p.Sequence != nil {
		atomic.StoreInt64(&g.sequence, *p.Sequence)
	}

	switch p.Op {
	case opHello:
		var hello struct {
			HeartbeatInterval int `json:"heartbeat_interval"`
		}
		if err := json.Unmarshal(p.Data, &hello); err != nil {
			return err
		}
		g.heartbeatInterval = time.Duration(hello.HeartbeatInterval) * time.Millisecond

		// Kick off the heartbeat loop.
		go g.heartbeatLoop()

		// Identify or resume.
		if g.sessionID != "" {
			return g.sendResume()
		}
		return g.sendIdentify()

	case opHeartbeatACK:
		g.lastACK.Store(true)

	case opHeartbeat:
		// Discord requested an out-of-band heartbeat.
		return g.sendHeartbeat()

	case opReconnect:
		// Discord wants us to reconnect and resume.
		log.Printf("[gateway] received Reconnect — closing for resume")
		g.conn.Close()
		return io.EOF

	case opInvalidSession:
		// d == false means the session is not resumable.
		var resumable bool
		_ = json.Unmarshal(p.Data, &resumable)
		if !resumable {
			g.sessionID = ""
			g.resumeURL = ""
			atomic.StoreInt64(&g.sequence, 0)
		}
		log.Printf("[gateway] InvalidSession (resumable=%v)", resumable)
		// Jitter before re-identifying.
		time.Sleep(time.Second)
		return g.sendIdentify()

	case opDispatch:
		g.handleDispatch(p)
	}

	return nil
}

// handleDispatch processes dispatch events (op 0).
func (g *gateway) handleDispatch(p *gatewayPayload) {
	switch p.Type {
	case "READY":
		var ready ReadyEvent
		if err := json.Unmarshal(p.Data, &ready); err != nil {
			return
		}
		g.sessionID = ready.SessionID
		g.resumeURL = ready.ResumeGatewayURL

		// Expose the bot's own User object.
		g.bot.mu.Lock()
		g.bot.self = &ready.User
		g.bot.mu.Unlock()

		log.Printf("[gateway] Ready — logged in as %s", ready.User.Tag())
	}

	// Forward to the event dispatcher.
	g.bot.events.dispatch(g.bot, p.Type, p.Data)
}

// ---------------------------------------------------------------------------
// Heartbeat
// ---------------------------------------------------------------------------

func (g *gateway) heartbeatLoop() {
	// Jitter: sleep a random fraction of the interval before the first beat.
	// This mirrors Discord's recommendation. We use a simple 0.5× jitter.
	jitter := g.heartbeatInterval / 2
	select {
	case <-time.After(jitter):
	case <-g.stopCh:
		return
	}

	ticker := time.NewTicker(g.heartbeatInterval)
	defer ticker.Stop()

	for {
		if err := g.sendHeartbeat(); err != nil {
			return
		}
		select {
		case <-ticker.C:
		case <-g.stopCh:
			return
		}
	}
}

func (g *gateway) sendHeartbeat() error {
	seq := atomic.LoadInt64(&g.sequence)
	var d interface{} = seq
	if seq == 0 {
		d = nil
	}
	return g.conn.WriteJSON(map[string]interface{}{
		"op": opHeartbeat,
		"d":  d,
	})
}

// ---------------------------------------------------------------------------
// Identify / Resume
// ---------------------------------------------------------------------------

func (g *gateway) sendIdentify() error {
	g.bot.mu.RLock()
	token := g.bot.token
	intents := g.bot.intents
	pres := g.bot.initialPresence
	g.bot.mu.RUnlock()

	payload := map[string]interface{}{
		"op": opIdentify,
		"d": map[string]interface{}{
			"token":   token,
			"intents": int(intents),
			"properties": map[string]string{
				"os":      "linux",
				"browser": "GoDiscord",
				"device":  "GoDiscord",
			},
			"presence": pres,
		},
	}
	return g.conn.WriteJSON(payload)
}

func (g *gateway) sendResume() error {
	g.bot.mu.RLock()
	token := g.bot.token
	g.bot.mu.RUnlock()

	return g.conn.WriteJSON(map[string]interface{}{
		"op": opResume,
		"d": map[string]interface{}{
			"token":      token,
			"session_id": g.sessionID,
			"seq":        atomic.LoadInt64(&g.sequence),
		},
	})
}

// ---------------------------------------------------------------------------
// Presence update
// ---------------------------------------------------------------------------

func (g *gateway) updatePresence(p presence) error {
	return g.conn.WriteJSON(map[string]interface{}{
		"op": opPresenceUpdate,
		"d":  p,
	})
}
