package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/coder/websocket"
)

var ErrHubCreationRateLimited = errors.New("hub creation rate limited")

// Client is one user connected to a hub for the lifetime of its WebSocket.
type Client struct {
	ID   string
	Conn *websocket.Conn
}

// Hub owns the clients currently connected to one game session.
type Hub struct {
	ID         string
	AccessCode string
	CreatedAt  time.Time

	mu      sync.RWMutex
	clients map[string]*Client
}

func (h *Hub) AddClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.ID] = client
}

func (h *Hub) RemoveClient(clientID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, clientID)
	return len(h.clients) == 0
}

func (h *Hub) Clients() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	return clients
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

type creationWindow struct {
	started time.Time
	count   int
}

// HubManager safely owns all live hubs and removes idle or expired ones.
type HubManager struct {
	mu              sync.RWMutex
	hubs            map[string]*Hub
	creationWindows map[string]creationWindow
	maxHubAge       time.Duration
	rateWindow      time.Duration
	rateLimit       int
	stopJanitor     chan struct{}
	doneJanitor     chan struct{}
}

func NewHubManager(maxHubAge, rateWindow time.Duration, rateLimit int) *HubManager {
	if maxHubAge <= 0 {
		maxHubAge = time.Hour
	}
	if rateWindow <= 0 {
		rateWindow = time.Minute
	}
	if rateLimit <= 0 {
		rateLimit = 5
	}

	m := &HubManager{
		hubs:            make(map[string]*Hub),
		creationWindows: make(map[string]creationWindow),
		maxHubAge:       maxHubAge,
		rateWindow:      rateWindow,
		rateLimit:       rateLimit,
		stopJanitor:     make(chan struct{}),
		doneJanitor:     make(chan struct{}),
	}
	go m.runJanitor()
	return m
}

func (m *HubManager) CreateHub(creator string) (*Hub, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()

	window := m.creationWindows[creator]
	if now.Sub(window.started) >= m.rateWindow {
		window = creationWindow{started: now}
	}
	if window.count >= m.rateLimit {
		return nil, ErrHubCreationRateLimited
	}
	window.count++
	m.creationWindows[creator] = window

	hub := &Hub{
		ID:         randomID(),
		AccessCode: randomID(),
		CreatedAt:  now,
		clients:    make(map[string]*Client),
	}
	m.hubs[hub.ID] = hub
	return hub, nil
}

func (m *HubManager) Hub(id string) (*Hub, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	hub, ok := m.hubs[id]
	return hub, ok
}

func (m *HubManager) RemoveClient(hubID, clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	hub, ok := m.hubs[hubID]
	if ok && hub.RemoveClient(clientID) {
		delete(m.hubs, hubID)
	}
}

func (m *HubManager) Close() {
	close(m.stopJanitor)
	<-m.doneJanitor
}

func (m *HubManager) runJanitor() {
	interval := m.maxHubAge / 10 
	if interval > time.Minute {
		interval = time.Minute
	}
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(m.doneJanitor)
	for {
		select {
		case <-ticker.C:
			m.deleteExpired(time.Now())
		case <-m.stopJanitor:
			return
		}
	}
}

// Delete all hubs that either have a connected client count of 0 or are past their max age. 
func (m *HubManager) deleteExpired(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, hub := range m.hubs {
		if hub.ClientCount() == 0 || now.Sub(hub.CreatedAt) >= m.maxHubAge {
			delete(m.hubs, id)
		}
	}
	for creator, window := range m.creationWindows {
		if now.Sub(window.started) >= m.rateWindow {
			delete(m.creationWindows, creator)
		}
	}
}

// Generate a random ID for a hub 
func randomID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
