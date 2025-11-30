package dht

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/anacrolix/dht/v2"
)

// Client wraps the anacrolix DHT client with libreseed-specific functionality
type Client struct {
	server  *dht.Server
	config  *ClientConfig
	mu      sync.RWMutex
	started bool
	ctx     context.Context
	cancel  context.CancelFunc
	stats   ClientStats
	nodeID  [20]byte
}

// ClientConfig holds DHT client configuration
type ClientConfig struct {
	// Port to listen on for DHT traffic
	Port int

	// Bootstrap nodes to initially connect to
	BootstrapNodes []string

	// NodeID for this DHT node (optional, auto-generated if empty)
	NodeID [20]byte

	// AnnounceInterval for periodic re-announcement
	AnnounceInterval time.Duration
}

// ClientStats tracks DHT client statistics
type ClientStats struct {
	mu                  sync.RWMutex
	NodesInRoutingTable int
	TotalQueries        uint64
	TotalResponses      uint64
	TotalAnnounces      uint64
	TotalLookups        uint64
	LastBootstrap       time.Time
}

// DefaultClientConfig returns a ClientConfig with sensible defaults
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Port: 6881,
		BootstrapNodes: []string{
			"router.bittorrent.com:6881",
			"dht.transmissionbt.com:6881",
			"router.utorrent.com:6881",
		},
		AnnounceInterval: 30 * time.Minute,
	}
}

// NewClient creates a new DHT client
func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		config = DefaultClientConfig()
	}

	// Generate node ID if not provided
	nodeID := config.NodeID
	if nodeID == [20]byte{} {
		// Generate random node ID
		// In production, this could be derived from the keypair
		copy(nodeID[:], []byte("libreseed-node-00000")[:20])
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		config: config,
		ctx:    ctx,
		cancel: cancel,
		nodeID: nodeID,
	}

	return client, nil
}

// Start initializes and starts the DHT client
func (c *Client) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return fmt.Errorf("DHT client already started")
	}

	// Create UDP connection for DHT
	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", c.config.Port))
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}

	// Create DHT server configuration
	serverConfig := dht.ServerConfig{
		Conn:       conn,
		NoSecurity: false,
		StartingNodes: func() ([]dht.Addr, error) {
			return c.resolveBootstrapNodes()
		},
	}

	// Create DHT server
	server, err := dht.NewServer(&serverConfig)
	if err != nil {
		return fmt.Errorf("failed to create DHT server: %w", err)
	}

	c.server = server
	c.started = true

	// Start background tasks
	go c.periodicTasks()

	// Initial bootstrap
	go c.bootstrap()

	return nil
}

// Stop gracefully shuts down the DHT client
func (c *Client) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	// Cancel background tasks
	c.cancel()

	// Close DHT server
	if c.server != nil {
		c.server.Close()
	}

	c.started = false
	return nil
}

// Announce announces a package to the DHT
func (c *Client) Announce(infoHash [20]byte, port int) error {
	c.mu.RLock()
	if !c.started {
		c.mu.RUnlock()
		return fmt.Errorf("DHT client not started")
	}
	server := c.server
	c.mu.RUnlock()

	// Announce to DHT
	_, err := server.Announce(infoHash, port, false)
	if err != nil {
		return fmt.Errorf("failed to announce: %w", err)
	}

	// Update statistics
	c.stats.mu.Lock()
	c.stats.TotalAnnounces++
	c.stats.mu.Unlock()

	return nil
}

// GetPeers queries the DHT for peers seeding a package
func (c *Client) GetPeers(infoHash [20]byte) ([]net.Addr, error) {
	c.mu.RLock()
	if !c.started {
		c.mu.RUnlock()
		return nil, fmt.Errorf("DHT client not started")
	}
	server := c.server
	c.mu.RUnlock()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	// Query DHT for peers
	peers := make([]net.Addr, 0)
	announce, err := server.Announce(infoHash, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to query peers: %w", err)
	}
	defer announce.Close()

	// Collect peers with timeout
	for {
		select {
		case peerValues, ok := <-announce.Peers:
			if !ok {
				// Channel closed, return collected peers
				c.stats.mu.Lock()
				c.stats.TotalLookups++
				c.stats.mu.Unlock()
				return peers, nil
			}
			// Extract peer addresses from PeersValues
			for _, peer := range peerValues.Peers {
				peers = append(peers, peer.UDP())
			}
		case <-ctx.Done():
			// Timeout reached
			c.stats.mu.Lock()
			c.stats.TotalLookups++
			c.stats.mu.Unlock()
			return peers, nil
		}
	}
}

// GetStats returns a snapshot of DHT statistics
func (c *Client) GetStats() ClientStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	// Get routing table size
	c.mu.RLock()
	if c.server != nil {
		c.stats.NodesInRoutingTable = c.server.NumNodes()
	}
	c.mu.RUnlock()

	return ClientStats{
		NodesInRoutingTable: c.stats.NodesInRoutingTable,
		TotalQueries:        c.stats.TotalQueries,
		TotalResponses:      c.stats.TotalResponses,
		TotalAnnounces:      c.stats.TotalAnnounces,
		TotalLookups:        c.stats.TotalLookups,
		LastBootstrap:       c.stats.LastBootstrap,
	}
}

// NodeID returns the client's DHT node ID
func (c *Client) NodeID() [20]byte {
	return c.nodeID
}

// IsStarted returns whether the client is running
func (c *Client) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// bootstrap connects to bootstrap nodes
func (c *Client) bootstrap() error {
	c.mu.RLock()
	server := c.server
	c.mu.RUnlock()

	if server == nil {
		return fmt.Errorf("DHT server not initialized")
	}

	// Resolve bootstrap nodes
	nodes, err := c.resolveBootstrapNodes()
	if err != nil {
		return fmt.Errorf("failed to resolve bootstrap nodes: %w", err)
	}

	// Bootstrap from nodes
	for _, node := range nodes {
		// Ping each bootstrap node
		go func(addr dht.Addr) {
			ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
			defer cancel()

			result := server.Ping(&net.UDPAddr{
				IP:   addr.IP(),
				Port: addr.Port(),
			})
			if result.Err == nil {
				c.stats.mu.Lock()
				c.stats.TotalQueries++
				c.stats.TotalResponses++
				c.stats.mu.Unlock()
			}

			<-ctx.Done()
		}(node)
	}

	// Update last bootstrap time
	c.stats.mu.Lock()
	c.stats.LastBootstrap = time.Now()
	c.stats.mu.Unlock()

	return nil
}

// resolveBootstrapNodes resolves bootstrap node addresses
func (c *Client) resolveBootstrapNodes() ([]dht.Addr, error) {
	addrs := make([]dht.Addr, 0)

	for _, node := range c.config.BootstrapNodes {
		// Resolve DNS if needed
		host, port, err := net.SplitHostPort(node)
		if err != nil {
			continue
		}

		ips, err := net.LookupIP(host)
		if err != nil {
			continue
		}

		for _, ip := range ips {
			// Only use IPv4 for now
			if ip.To4() != nil {
				udpAddr := &net.UDPAddr{
					IP:   ip.To4(),
					Port: parsePort(port),
				}
				addr := dht.NewAddr(udpAddr)
				addrs = append(addrs, addr)
				break // Only use first IPv4 address per host
			}
		}
	}

	return addrs, nil
}

// periodicTasks runs background maintenance tasks
func (c *Client) periodicTasks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Re-bootstrap periodically to maintain routing table
			if time.Since(c.stats.LastBootstrap) > 30*time.Minute {
				c.bootstrap()
			}
		}
	}
}

// parsePort parses a port string to integer
func parsePort(portStr string) int {
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return port
}
