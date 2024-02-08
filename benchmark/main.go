package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	BaseURLDriver  = "localhost:8081"
	BaseURLTracker = "localhost:8080"

	// call type
	Driver  = 1
	Tracker = 2

	// metric key
	Total        = "total"
	Success      = "success"
	Failure      = "failure"
	ResponseTime = "response_time"
)

type Location struct {
	Long      float64 `json:"long"`
	Lat       float64 `json:"lat"`
	Timestamp string  `json:"timestamp"`
}

type Metric struct {
	mu            sync.Mutex
	metricSend    map[string]int
	metricReceive map[string]int
	responseTime  []float64
}

func (m *Metric) Success(actor int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch actor {
	case Driver:
		m.metricSend[Success] += 1
	case Tracker:
		m.metricReceive[Success] += 1
	}
}

func (m *Metric) Total(actor int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch actor {
	case Driver:
		m.metricSend[Total] += 1
	case Tracker:
		m.metricReceive[Total] += 1
	}
}

func (m *Metric) Failure(actor int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch actor {
	case Driver:
		m.metricSend[Failure] += 1
	case Tracker:
		m.metricReceive[Failure] += 1
	}
}

func (m *Metric) ResponseTime(i float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.responseTime = append(m.responseTime, i)
}

func median(data []float64) float64 {
	dataCopy := make([]float64, len(data))
	copy(dataCopy, data)

	sort.Float64s(dataCopy)

	var median float64
	l := len(dataCopy)
	if l == 0 {
		return 0
	} else if l%2 == 0 {
		median = (dataCopy[l/2-1] + dataCopy[l/2]) / 2
	} else {
		median = dataCopy[l/2]
	}

	return median
}

func main() {
	m := Metric{
		metricSend:    make(map[string]int),
		metricReceive: make(map[string]int),
	}

	defer func() {
		median := median(m.responseTime)
		log.Info().Any("metric send", m.metricSend).Any("metric receive", m.metricReceive).Any("response time", m.responseTime).Any("median", median).Msg("finished")
	}()

	ready := make(chan struct{}, 5)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		log.Info().Msg("starting request as tracker")
		ReceiveLocation(5, 1, &m, ready)
		log.Info().Msg("finished request as tracker")

	}()
	go func() {
		defer wg.Done()

		log.Info().Msg("all client connected")
		log.Info().Msg("starting request as driver")
		SendLocation(7, &m)
		log.Info().Msg("finished request as driver")
	}()

	wg.Wait()
}

// SendLocation will send location data as a driver
// and will calculate metric by message sent by driver.
// n will be the number of request will made
func SendLocation(n int, m *Metric) {
	for i := 0; i < n; i++ {
		m.Total(Driver)

		loc := Location{
			Long:      float64(i),
			Lat:       float64(i),
			Timestamp: time.Now().Format(time.RFC3339Nano),
		}

		b, err := json.Marshal(loc)
		if err != nil {
			log.Error().Err(err).Msg("error when marshaling location data")
			m.Failure(Driver)
		}
		body := bytes.NewReader(b)
		url := fmt.Sprintf("http://%s/%s?bus_id=1", BaseURLDriver, "location")
		resp, err := http.Post(url, "application/json", body)
		if err != nil {
			log.Error().Err(err).Msg("error when calling http request")
			m.Failure(Driver)
		}
		defer resp.Body.Close()

		var locResp Location
		if err = json.NewDecoder(resp.Body).Decode(&locResp); err != nil {
			log.Error().Err(err).Msg("error when unmarshaling location data")
			m.Failure(Driver)
		}

		if resp.StatusCode == http.StatusOK {
			m.Success(Driver)
		} else {
			log.Error().Any("status code", resp.Status).Msg("http call error")
			m.Failure(Driver)
		}
	}
}

// ReceiveLocation will receive location will receive location as a client
// and will calculate metric by message sent by server
// n will be number of concurrent user, i will be number of message that sent by each user
func ReceiveLocation(n, i int, m *Metric, ready chan struct{}) {
	var wg sync.WaitGroup
	wg.Add(n)
	for usr := 0; usr < n; usr++ {
		go func() {
			log.Info().Any("user", usr).Msg("starting connection")
			defer wg.Done()
			m.Total(Tracker)

			url := fmt.Sprintf("ws://%s/%s", BaseURLTracker, "location")
			header := http.Header{}
			header.Add("Content-Type", "application/json")
			dial, _, err := websocket.DefaultDialer.Dial(url, header)
			if err != nil {
				log.Error().Err(err).Msg("error when dialing websocket")
				m.Failure(Tracker)
			}
			defer dial.Close()
			log.Info().Any("user", usr).Msg("user connected")
			ready <- struct{}{}

			for resp := 0; resp < i; resp++ {
				var locResp Location
				dial.SetReadDeadline(time.Now().Add(10 * time.Second))
				if err = dial.ReadJSON(&locResp); err != nil {
					log.Error().Err(err).Msg("error when read location response")
					m.Failure(Tracker)
					break
				}

				ts, err := time.Parse(time.RFC3339Nano, locResp.Timestamp)
				if err != nil {
					log.Error().Err(err).Msg("error when parse location timestamp")
					m.Failure(Tracker)
					break
				}

				m.Success(Tracker)
				m.ResponseTime(time.Since(ts).Seconds())
			}

		}()
	}
	wg.Wait()
}
