package kafka

import (
	"context"
	"encoding/json"

	"github.com/rafimuhammad01/tracking-app/track"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type Handler struct {
	r Receiver
}

type Receiver interface {
	Receive(l track.Location)
}

func (h *Handler) Listen(ctx context.Context, r *kafka.Reader) {
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to read message")
		}
		log.Debug().Any("key", m).Msg("message received")

		var loc track.Location
		json.Unmarshal(m.Value, &loc)

		h.r.Receive(loc)
	}
}

func NewHandler(r Receiver) *Handler {
	return &Handler{
		r: r,
	}
}
