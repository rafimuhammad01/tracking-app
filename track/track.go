package track

import "sync"

// Customer denotes the customers object.
// For now, we will just need the ID of it,
// in next iteration we probably need to adjust it for better representing customer
type Customer struct {
	ID string
}

// Location denotes the location object.
type Location struct {
	Long float64
	Lat  float64
}

// Hub will be saving all the customer that connects with our apps.
// Hub will save the customer with location channel.
// Location channel must be defined by the client and client will also need to handle the event sent to that channel.
type Hub struct {
	customers map[string]chan Location
	mu        sync.Mutex
}

// Register will register the client to the Hub.
// If client want to receive message they need to register the customer and location channel.
func (h *Hub) Register(c Customer, l chan Location) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.customers[c.ID] = l
}

// Unregister will remove the client from the Hub and also close their registered channel
func (h *Hub) Unregister(c Customer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ch, ok := h.customers[c.ID]; ok {
		close(ch)
	}

	delete(h.customers, c.ID)
}

// Receive will be receiving the location and send that location to all registered customer.
func (h *Hub) Receive(l Location) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, v := range h.customers {
		v <- l
	}
}

// NewHub will initiate Hub property.
func NewHub() *Hub {
	return &Hub{
		customers: make(map[string]chan Location),
	}
}
