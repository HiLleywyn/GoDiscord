package discord

// gateway.go — Discord Gateway v10 client.
//
// Responsibilities:
//   - Establish a TLS WebSocket connection to the Discord Gateway.
//   - Send the Identify payload to authenticate.
//   - Maintain the heartbeat loop required by Discord.
//   - Dispatch incoming events to the event dispatcher.
//   - Automatically reconnect and resume a session after disconnects,
//     using exponential back-off with jitter.
//   - Detect zombie connections (missed heartbeat ACKs) and force reconnect.

import (
	"encoding/json"
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Discord Gateway opcodes.
// https://discord.com/developers/docs/topics/opcodes-and-status-codes
const (
	opDispatch       = 0
	opHeartbeat      = 1
	opIdentify       = 2
	opPresenceUpdate = 3
	opResume         = 6
	opReconnect      = 7
	opInvalidSession = 9
	opHello          = 10
	opHeartbeatACK   = 11
)

const (
	gatewayURL = "wss://gateway.discord.gg/?v=10&encoding=json"
)

// Reconnect back-off parameters.
const (
	backoffBase   = time.Second      // initial wait
	backoffMax    = 5 * time.Minute  // cap
	backoffFactor = 2.0              // multiplier per attempt
	backoffJitter = 0.2              // ±20 % random jitter
)

// gateway manages the Discord Gateway connection for a Bot.
type gateway struct {
	bot *Bot

	// Connection state — protected by sessionMu.
	conn      *wsConn
	sessionMu sync.RWMutex
	sessionID string
	resumeURL string

	// sequence is the last seen dispatch sequence number (atomic).
	sequence int64

	// Heartbeat state.
	heartbeatInterval time.Duration
	lastACK           atomic.Bool // set to true on ACK, false before each send

	// Lifecycle channels.
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

// start performs the first connection synchronously, then launches the
// reconnect loop in a goroutine. Returns immediately after the initial
// connection succeeds or fails.
func (g *gateway) start() error {
	if err := g.connect(); err != nil {
		return err
	}
	go g.loop()
	return nil
}

// stop signals the gateway to disconnect and waits for the read-loop to exit.
func (g *gateway) stop() {
	close(g.stopCh)
	g.sessionMu.RLock()
	conn := g.conn
	g.sessionMu.RUnlock()
	if conn != nil {
		conn.Close()
	}
	<-g.doneCh
}

// loop drives reconnection with exponential back-off. It runs until stop() is
// called.
func (g *gateway) loop() {
	defer close(g.doneCh)

	attempt := 0

	for {
		err := g.readLoop()

		select {
		case <-g.stopCh:
			return
		default:
		}

		delay := backoffDelay(attempt)
		if err != nil && err != io.EOF {
			g.bot.log.Printf("[gateway] disconnected: %v — reconnecting in %s", err, delay.Round(time.Millisecond))
		} else {
			g.bot.log.Printf("[gateway] disconnected — reconnecting in %s", delay.Round(time.Millisecond))
		}

		select {
		case <-time.After(delay):
		case <-g.stopCh:
			return
		}

		if err := g.connect(); err != nil {
			g.bot.log.Printf("[gateway] reconnect failed: %v", err)
			attempt++
			continue
		}
		attempt = 0 // reset on successful connection
	}
}

// backoffDelay computes the wait time for reconnect attempt n using
// exponential back-off with ±jitter%.
func backoffDelay(attempt int) time.Duration {
	d := backoffBase
	for i := 0; i < attempt; i++ {
		d = time.Duration(float64(d) * backoffFactor)
		if d > backoffMax {
			d = backoffMax
			break
		}
	}
	// Apply ±backoffJitter random jitter.
	jitter := 1 + (rand.Float64()*2-1)*backoffJitter
	return time.Duration(float64(d) * jitter)
}

// connect opens a new WebSocket connection and stores it. Does not send
// Identify or Resume — that happens in the Hello handler inside readLoop.
func (g *gateway) connect() error {
	g.sessionMu.RLock()
	resume := g.resumeURL
	g.sessionMu.RUnlock()

	target := gatewayURL
	if resume != "" {
		target = resume + "?v=10&encoding=json"
	}

	conn, err := wsDial(target)
	if err != nil {
		// Clear the resume URL so the next attempt falls back to the primary
		// gateway rather than retrying a potentially stale URL.
		g.sessionMu.Lock()
		g.resumeURL = ""
		g.sessionMu.Unlock()
		return err
	}

	g.sessionMu.Lock()
	g.conn = conn
	g.sessionMu.Unlock()
	return nil
}

// ---------------------------------------------------------------------------
// Read loop
// ---------------------------------------------------------------------------

// readLoop reads frames until an error or forced close.
func (g *gateway) readLoop() error {
	for {
		g.sessionMu.RLock()
		conn := g.conn
		g.sessionMu.RUnlock()

		data, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var p gatewayPayload
		if err := json.Unmarshal(data, &p); err != nil {
			g.bot.log.Printf("[gateway] unmarshal error: %v", err)
			continue
		}

		if err := g.handlePayload(&p); err != nil {
			return err
		}
	}
}

// handlePayload processes a single gateway payload.
func (g *gateway) handlePayload(p *gatewayPayload) error {
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

		// Prime lastACK so the first zombie check doesn't fire prematurely.
		g.lastACK.Store(true)

		go g.heartbeatLoop()

		g.sessionMu.RLock()
		sid := g.sessionID
		g.sessionMu.RUnlock()

		if sid != "" {
			return g.sendResume()
		}
		return g.sendIdentify()

	case opHeartbeatACK:
		g.lastACK.Store(true)

	case opHeartbeat:
		return g.sendHeartbeat()

	case opReconnect:
		g.bot.log.Printf("[gateway] received Reconnect — closing for resume")
		g.sessionMu.RLock()
		conn := g.conn
		g.sessionMu.RUnlock()
		conn.Close()
		return io.EOF

	case opInvalidSession:
		var resumable bool
		_ = json.Unmarshal(p.Data, &resumable)
		if !resumable {
			g.sessionMu.Lock()
			g.sessionID = ""
			g.resumeURL = ""
			g.sessionMu.Unlock()
			atomic.StoreInt64(&g.sequence, 0)
		}
		g.bot.log.Printf("[gateway] InvalidSession (resumable=%v)", resumable)
		// Discord recommends a random 1–5 s jitter before re-identifying.
		jitter := time.Duration(1000+rand.Intn(4000)) * time.Millisecond
		time.Sleep(jitter)
		return g.sendIdentify()

	case opDispatch:
		g.handleDispatch(p)
	}

	return nil
}

// handleDispatch processes dispatch events (op 0).
func (g *gateway) handleDispatch(p *gatewayPayload) {
	if p.Type == "READY" {
		var ready ReadyEvent
		if err := json.Unmarshal(p.Data, &ready); err != nil {
			g.bot.log.Printf("[gateway] failed to unmarshal READY: %v", err)
			return
		}

		g.sessionMu.Lock()
		g.sessionID = ready.SessionID
		g.resumeURL = ready.ResumeGatewayURL
		g.sessionMu.Unlock()

		g.bot.mu.Lock()
		g.bot.self = &ready.User
		g.bot.mu.Unlock()

		g.bot.log.Printf("[gateway] Ready — logged in as %s", ready.User.Tag())
	}

	g.bot.events.dispatch(g.bot, p.Type, p.Data)
}

// ---------------------------------------------------------------------------
// Heartbeat
// ---------------------------------------------------------------------------

// heartbeatLoop sends heartbeats on the Discord-specified interval.
// If a heartbeat ACK is not received before the next send, the connection is
// treated as a zombie and closed to trigger a resume.
func (g *gateway) heartbeatLoop() {
	// Initial jitter: sleep a random 0–interval fraction before the first beat.
	// This prevents thundering-herd on mass reconnects.
	jitter := time.Duration(rand.Int63n(int64(g.heartbeatInterval)))
	select {
	case <-time.After(jitter):
	case <-g.stopCh:
		return
	}

	ticker := time.NewTicker(g.heartbeatInterval)
	defer ticker.Stop()

	for {
		// Zombie detection: if we haven't received an ACK since the last
		// heartbeat, the connection is stale — close it to force a resume.
		if !g.lastACK.Load() {
			g.bot.log.Printf("[gateway] heartbeat ACK not received — zombie connection detected, reconnecting")
			g.sessionMu.RLock()
			conn := g.conn
			g.sessionMu.RUnlock()
			if conn != nil {
				conn.Close()
			}
			return
		}

		// Mark ACK as not yet received before sending the next heartbeat.
		g.lastACK.Store(false)

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
	g.sessionMu.RLock()
	conn := g.conn
	g.sessionMu.RUnlock()
	return conn.WriteJSON(map[string]interface{}{
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

	g.sessionMu.RLock()
	conn := g.conn
	g.sessionMu.RUnlock()

	return conn.WriteJSON(map[string]interface{}{
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
	})
}

func (g *gateway) sendResume() error {
	g.bot.mu.RLock()
	token := g.bot.token
	g.bot.mu.RUnlock()

	g.sessionMu.RLock()
	sid := g.sessionID
	conn := g.conn
	g.sessionMu.RUnlock()

	return conn.WriteJSON(map[string]interface{}{
		"op": opResume,
		"d": map[string]interface{}{
			"token":      token,
			"session_id": sid,
			"seq":        atomic.LoadInt64(&g.sequence),
		},
	})
}

// ---------------------------------------------------------------------------
// Presence update
// ---------------------------------------------------------------------------

func (g *gateway) updatePresence(p presence) error {
	g.sessionMu.RLock()
	conn := g.conn
	g.sessionMu.RUnlock()
	if conn == nil {
		return nil
	}
	return conn.WriteJSON(map[string]interface{}{
		"op": opPresenceUpdate,
		"d":  p,
	})
}
