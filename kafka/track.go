package kafka

import (
	"context"
	"encoding/json"
	"io"

	"github.com/rafimuhammad01/tracking-app/track"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type Tracker struct {
	receiver Receiver

	r *kafka.Reader
	w *kafka.Writer
}

type Receiver interface {
	Receive(l track.Location)
}

func (t *Tracker) Listen(ctx context.Context) {
	for {
		m, err := t.r.ReadMessage(ctx)
		if err != nil {
			if err == io.EOF {
				log.Info().Msg("connection closed")
				break
			}
			log.Error().Err(err).Msg("failed to read message")
			continue
		}
		log.Debug().Any("key", m).Msg("message received")

		var loc track.Location
		err = json.Unmarshal(m.Value, &loc)
		if err != nil {
			log.Error().Err(err).Msg("failed to unmarshal message")
		}

		t.receiver.Receive(loc)
	}
}

func (t *Tracker) Send(ctx context.Context, l track.Location) error {
	b, err := json.Marshal(l)
	if err != nil {
		log.Debug().Err(err).Msg("failed to marshal location")
		return err
	}

	err = t.w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(l.Bus.ID),
		Value: b,
	})
	if err != nil {
		log.Debug().Err(err).Msg("failed to marshal location")
		return err
	}

	return nil
}

func (t *Tracker) ReaderCloser() error {
	if err := t.r.Close(); err != nil {
		return err
	}
	return nil
}

func (t *Tracker) ReaderTopic() kafka.ReaderConfig {
	return t.r.Config()
}

type opts func(*Tracker)

func NewTracker(opts ...opts) *Tracker {
	t := Tracker{}

	for _, opt := range opts {
		opt(&t)
	}

	return &t
}

func WithReceiver(rc Receiver, r *kafka.Reader) opts {
	return func(t *Tracker) {
		t.receiver = rc
		t.r = r
	}
}

func WithWriter(w *kafka.Writer) opts {
	return func(t *Tracker) {
		t.w = w
	}
}
