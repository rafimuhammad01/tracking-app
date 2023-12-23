package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rafimuhammad01/tracking-app/track"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{} // use default options

type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

type TrackingHandler struct {
	trackingSvc TrackingService
}

type TrackingService interface {
	Send(ctx context.Context, l track.Location) error
	Register(c track.Customer, l chan track.Location)
	Unregister(c track.Customer)
}

func (s *TrackingHandler) GetLatestLocation(w http.ResponseWriter, r *http.Request) {
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
	log.Debug().Any("customer", customer).Msg("client registered")

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
			locResp := struct {
				Long      float64 `json:"long"`
				Lat       float64 `json:"lat"`
				Timestamp string  `json:"timestamp"`
			}{
				Long:      l.Long,
				Lat:       l.Lat,
				Timestamp: l.Timestamp.Format(time.RFC3339Nano),
			}
			err = c.WriteJSON(locResp)
			if err != nil {
				log.Error().Err(err).Msg("websocket write json error")
				return
			}
		}
	}
}

func (d *TrackingHandler) SendLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Error: "method not allowed"})
		return
	}

	// parse location data.
	var loc track.Location
	var locReq struct {
		Long      float64 `json:"long"`
		Lat       float64 `json:"lat"`
		Timestamp string  `json:"timestamp"`
	}
	err := json.NewDecoder(r.Body).Decode(&locReq)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Error: "invalid request body"})
		return
	}
	loc.Long = locReq.Long
	loc.Lat = locReq.Lat

	// parse vehicle data
	busID := r.URL.Query().Get("bus_id")
	if busID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Error: "invalid bus_id value"})
		return
	}
	loc.Bus.ID = busID

	// parse timestamp
	loc.Timestamp = time.Now()
	if locReq.Timestamp != "" {
		ts, err := time.Parse(time.RFC3339Nano, locReq.Timestamp)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{Error: "invalid timestamp value"})
			return
		}

		loc.Timestamp = ts
	}

	// send location
	if err := d.trackingSvc.Send(r.Context(), loc); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Error: "internal server error"})
		return
	}

	// return
	json.NewEncoder(w).Encode(Response{Data: "success"})
	return
}

func NewHandler(trackingSvc TrackingService) *TrackingHandler {
	return &TrackingHandler{
		trackingSvc: trackingSvc,
	}
}
