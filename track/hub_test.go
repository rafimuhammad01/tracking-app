package track

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTracking(t *testing.T) {
	h := hub{
		customers: make(map[string]chan Location),
	}

	var wg sync.WaitGroup
	var wgRegister sync.WaitGroup
	var mu sync.Mutex
	msgReceived := make(map[string][]Location)

	// simulate 3 request that register user
	wg.Add(3)
	wgRegister.Add(3) // this wg to wait for the client to register first.
	for i := 0; i < 3; i++ {
		go func(id string) {
			c := Customer{ID: id}
			l := make(chan Location)
			h.register(c, l)
			wgRegister.Done()

			for j := 0; j < 3; j++ {
				loc := <-l
				mu.Lock()
				msgReceived[id] = append(msgReceived[id], loc)
				mu.Unlock()
			}

			h.unregister(c)
			wg.Done()
		}(fmt.Sprint(i))
	}

	// simulate listener that receive location
	wgRegister.Wait()
	wg.Add(1)
	go func() {
		for i := 0; i < 3; i++ {
			loc := Location{
				Long: float64(i),
				Lat:  float64(i),
			}
			h.receive(loc)
		}
		wg.Done()
	}()

	wg.Wait()
	expected := map[string][]Location{
		"0": {Location{0, 0, Bus{}}, Location{1, 1, Bus{}}, Location{2, 2, Bus{}}},
		"1": {Location{0, 0, Bus{}}, Location{1, 1, Bus{}}, Location{2, 2, Bus{}}},
		"2": {Location{0, 0, Bus{}}, Location{1, 1, Bus{}}, Location{2, 2, Bus{}}},
	}

	assert.EqualValues(t, expected, msgReceived)
	assert.Empty(t, h.customers)
}
