package draw

import (
	"fmt"
	"log"
)

// DrawServer manages the draw protocol server
type DrawServer struct {
	screen  *Screen
	clients map[int]*DrawClient
	nextID  int
	running bool
}

// NewDrawServer creates a new draw server
func NewDrawServer(screen *Screen) *DrawServer {
	return &DrawServer{
		screen:  screen,
		clients: make(map[int]*DrawClient),
		nextID:  1,
		running: false,
	}
}

// Start starts the draw server
func (s *DrawServer) Start() error {
	s.running = true
	log.Printf("Draw server started")
	return nil
}

// Stop stops the draw server
func (s *DrawServer) Stop() error {
	s.running = false
	s.clients = make(map[int]*DrawClient)
	return nil
}

// NewClient creates a new client connection
func (s *DrawServer) NewClient() *DrawClient {
	client := NewDrawClient(s.screen)
	s.clients[s.nextID] = client
	s.nextID++
	return client
}

// RemoveClient removes a client connection
func (s *DrawServer) RemoveClient(id int) {
	delete(s.clients, id)
}

// HandleCommand handles a draw command from a client
func (s *DrawServer) HandleCommand(clientID int, buf []byte) (int, error) {
	client := s.clients[clientID]
	if client == nil {
		return 0, fmt.Errorf("unknown client: %d", clientID)
	}

	return client.HandleCommand(buf)
}

// GetScreen returns the server's screen
func (s *DrawServer) GetScreen() *Screen {
	return s.screen
}

// Flush flushes the screen to the display
func (s *DrawServer) Flush() error {
	if s.screen != nil {
		return s.screen.Flush()
	}
	return nil
}
