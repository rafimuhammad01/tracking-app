package track

import "sync"

// hub will be saving all the customer that connects with our apps.
// hub will save the customer with location channel.
// Location channel must be defined by the client and client will also need to handle the event sent to that channel.
type hub struct {
	customers map[string]chan Location
	mu        sync.Mutex
}

func (h *hub) register(c Customer, l chan Location) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.customers[c.ID] = l
}

func (h *hub) unregister(c Customer) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if ch, ok := h.customers[c.ID]; ok {
		close(ch)
	}

	delete(h.customers, c.ID)
}

func (h *hub) receive(l Location) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, v := range h.customers {
		v <- l
	}
}
