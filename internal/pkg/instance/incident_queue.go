package instance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

type IncidentQueue struct {
	lock        sync.Mutex
	items       []Incident
	persistPath string
}

func NewIncidentQueue(instancePath string) *IncidentQueue {
	persistFile := ""
	if instancePath != "" {
		persistFile = filepath.Join(instancePath, "incident_queue.json")
	}

	q := &IncidentQueue{
		items:       make([]Incident, 0, 50),
		persistPath: persistFile,
	}

	// Load any unflushed incidents from previous run / crash
	q.LoadFromDisk()

	return q
}

func (q *IncidentQueue) LoadFromDisk() {
	if q.persistPath == "" {
		return
	}

	data, err := os.ReadFile(q.persistPath)
	if err != nil || len(data) == 0 {
		return
	}

	var savedItems []Incident
	if err := json.Unmarshal(data, &savedItems); err == nil && len(savedItems) > 0 {
		q.lock.Lock()
		q.items = append(savedItems, q.items...)
		q.lock.Unlock()

		logrus.WithFields(logrus.Fields{
			"count": len(savedItems),
			"file":  q.persistPath,
		}).Info("Restored unflushed incidents from persistent disk queue after restart/crash")

		// Truncate file now that items are loaded into memory
		_ = os.Remove(q.persistPath)
	}
}

func (q *IncidentQueue) SaveToDisk() {
	if q.persistPath == "" {
		return
	}

	q.lock.Lock()
	defer q.lock.Unlock()

	if len(q.items) == 0 {
		_ = os.Remove(q.persistPath)
		return
	}

	data, err := json.MarshalIndent(q.items, "", "  ")
	if err != nil {
		logrus.WithError(err).Error("Failed to serialize incident queue to disk")
		return
	}

	if err := os.WriteFile(q.persistPath, data, 0644); err != nil {
		logrus.WithError(err).Error("Failed to write persistent incident queue to disk")
	} else {
		logrus.WithFields(logrus.Fields{
			"count": len(q.items),
			"file":  q.persistPath,
		}).Info("Persisted unflushed incident queue to disk")
	}
}

func (q *IncidentQueue) Enqueue(inc Incident) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.items = append(q.items, inc)
}

func (q *IncidentQueue) DequeueBatch(maxBatch int) []Incident {
	q.lock.Lock()
	defer q.lock.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	n := maxBatch
	if n > len(q.items) {
		n = len(q.items)
	}

	batch := make([]Incident, n)
	copy(batch, q.items[:n])
	q.items = q.items[n:]

	return batch
}

func (q *IncidentQueue) Drain() []Incident {
	q.lock.Lock()
	defer q.lock.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	drained := q.items
	q.items = make([]Incident, 0, 50)
	return drained
}

func (q *IncidentQueue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()
	return len(q.items)
}
