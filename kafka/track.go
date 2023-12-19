package kafka

import (
	"context"
	"time"

	"github.com/rafimuhammad01/tracking-app/track"
)

type Handler struct {
	r Receiver
}

type Receiver interface {
	Receive(ctx context.Context, l track.Location)
}

func (h *Handler) Listen() {
	locTest := float64(0)
	ctx := context.Background()

	// TODO logic to parse location

	for {
		loc := track.Location{
			Long: locTest,
			Lat:  locTest,
		}

		h.r.Receive(ctx, loc)

		locTest++
		time.Sleep(time.Second)
	}
}

func NewHandler(r Receiver) *Handler {
	return &Handler{
		r: r,
	}
}
