package track

import "context"

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
	Bus  Bus
}

// Bus denotes the bus object
type Bus struct {
	ID string
}

// Sender will be the contract to send location
type Sender interface {
	Send(ctx context.Context, l Location) error
}

// Tracker will responsible for the tracking location including receiving location and sending location
type Tracker struct {
	h *hub
	s Sender
}

// Send will send the location and bus information
func (t *Tracker) Send(ctx context.Context, l Location) error {
	return t.s.Send(ctx, l)
}

// Receive will be receiving the location and send that location to all registered customer.
func (t *Tracker) Receive(l Location) {
	t.h.receive(l)
}

// Register will register the client to the Hub.
// If client want to receive message they need to register the customer and location channel.
func (t *Tracker) Register(c Customer, l chan Location) {
	t.h.register(c, l)
}

// Unregister will remove the client from the Hub and also close their registered channel
func (t *Tracker) Unregister(c Customer) {
	t.h.unregister(c)
}

type opts func(*Tracker)

// NewTracker will create new Tracker
func NewTracker(opts ...opts) *Tracker {
	t := Tracker{}

	for _, opt := range opts {
		opt(&t)
	}

	return &t
}

// WithHub will be initiate hub for receiving and broadcasting message
func WithHub() opts {
	return func(t *Tracker) {
		t.h = &hub{
			customers: make(map[string]chan Location),
		}
	}
}

// WithSender will assign sender to tracker and activate Tracker ability to send message
func WithSender(s Sender) opts {
	return func(t *Tracker) {
		t.s = s
	}
}
