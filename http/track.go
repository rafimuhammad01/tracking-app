package http

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rafimuhammad01/tracking-app/track"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{} // use default options

type Handler struct {
	trackingSvc TrackingService
}

type TrackingService interface {
	Register(c track.Customer, l chan track.Location)
	Unregister(c track.Customer)
}

func (s *Handler) GetLatestLocation(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("error when upgrade header")
		return
	}
	defer c.Close()

	// listen to disconnect event from client
	// ReadMessage will be return error if client is disconnect
	// WriteMessage won't
	errChan := make(chan error)
	go func() {
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				errChan <- err
			}
		}
	}()

	// setup customer
	locChan := make(chan track.Location)
	id := r.Header.Get("Session-ID")
	customer := track.Customer{ID: id}

	// register customer so we can track
	s.trackingSvc.Register(customer, locChan)
	log.Info().Any("customer", customer).Msg("client registered")

	// listen to location channel
	for {
		select {
		case err := <-errChan:
			s.trackingSvc.Unregister(customer)
			if websocket.IsCloseError(err, websocket.CloseNoStatusReceived) {
				log.Info().Msg("connection closed by client")
				return
			}
			log.Error().Err(err).Msg("websocket connection error")
			return
		case l := <-locChan:
			err = c.WriteJSON(l)
			if err != nil {
				log.Error().Err(err).Msg("websocket write json error")
				return
			}
		}
	}
}

func NewHandler(trackingSvc TrackingService) *Handler {
	return &Handler{
		trackingSvc: trackingSvc,
	}
}
