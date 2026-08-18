package instance

import (
	"sync"
	"time"
)

type TelemetryBuffer struct {
	lock          sync.RWMutex
	retention     time.Duration
	buffersByCar  map[int][]TelemetryFrame
	maxCarHistory int
}

func NewTelemetryBuffer(retention time.Duration) *TelemetryBuffer {
	if retention <= 0 {
		retention = 5 * time.Second
	}
	return &TelemetryBuffer{
		retention:     retention,
		buffersByCar:  make(map[int][]TelemetryFrame),
		maxCarHistory: 150, // Approx ~15s at 10Hz to strictly bound memory
	}
}

func (b *TelemetryBuffer) Push(frame TelemetryFrame) {
	b.lock.Lock()
	defer b.lock.Unlock()

	history := b.buffersByCar[frame.CarID]
	history = append(history, frame)

	// Prune frames older than retention window or exceeding max count
	cutoff := frame.ReceivedAt.Add(-b.retention)
	startIdx := 0
	for i, f := range history {
		if f.ReceivedAt.After(cutoff) {
			startIdx = i
			break
		}
	}

	if startIdx > 0 {
		history = history[startIdx:]
	}

	if len(history) > b.maxCarHistory {
		history = history[len(history)-b.maxCarHistory:]
	}

	b.buffersByCar[frame.CarID] = history
}

func (b *TelemetryBuffer) GetHistory(carID int) []TelemetryFrame {
	b.lock.RLock()
	defer b.lock.RUnlock()

	history, ok := b.buffersByCar[carID]
	if !ok || len(history) == 0 {
		return nil
	}

	result := make([]TelemetryFrame, len(history))
	copy(result, history)
	return result
}

func (b *TelemetryBuffer) GetLatest(carID int) (TelemetryFrame, bool) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	history, ok := b.buffersByCar[carID]
	if !ok || len(history) == 0 {
		return TelemetryFrame{}, false
	}

	return history[len(history)-1], true
}

func (b *TelemetryBuffer) Clear() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.buffersByCar = make(map[int][]TelemetryFrame)
}
