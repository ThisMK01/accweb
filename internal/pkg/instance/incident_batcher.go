package instance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

type IncidentBatcher struct {
	collector     *TelemetryCollector
	queue         *IncidentQueue
	maxBatchSize  int
	flushInterval time.Duration
	httpClient    *http.Client
	stopChan      chan struct{}
}

func NewIncidentBatcher(collector *TelemetryCollector, queue *IncidentQueue) *IncidentBatcher {
	return &IncidentBatcher{
		collector:     collector,
		queue:         queue,
		maxBatchSize:  10,
		flushInterval: 3 * time.Second,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

func (b *IncidentBatcher) Start() {
	go b.run()
}

func (b *IncidentBatcher) Stop() {
	close(b.stopChan)
	b.Flush()
}

func (b *IncidentBatcher) run() {
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopChan:
			return
		case <-ticker.C:
			if b.queue.Len() > 0 {
				b.flushBatch()
			}
		}
	}
}

func (b *IncidentBatcher) Flush() {
	for b.queue.Len() > 0 {
		b.flushBatch()
	}
}

func (b *IncidentBatcher) flushBatch() {
	incidents := b.queue.DequeueBatch(b.maxBatchSize)
	if len(incidents) == 0 {
		return
	}

	eventId := b.collector.eventId
	serverId := b.collector.instance.GetID()

	if eventId == "" {
		logrus.WithField("count", len(incidents)).Warn("Event ID not set; dropping or skipping incident flush")
		return
	}

	payload := IncidentBatchPayload{
		EventID:   eventId,
		ServerID:  serverId,
		SessionID: "live_session",
		Incidents: incidents,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		logrus.WithError(err).Error("Failed to serialize incident batch payload")
		return
	}

	url := fmt.Sprintf("%s/events/%s/incidents/batch", b.collector.backendApiUrl, eventId)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		logrus.WithError(err).Error("Failed to create incident batch HTTP request")
		b.requeue(incidents)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		logrus.WithError(err).Warn("Failed to send incident batch to backend; will retry")
		b.requeue(incidents)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logrus.WithFields(logrus.Fields{
			"event_id": eventId,
			"count":    len(incidents),
		}).Info("Successfully flushed incident batch to backend")
	} else {
		logrus.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"event_id":    eventId,
			"count":       len(incidents),
		}).Warn("Backend returned non-2xx status for incident batch")
		b.requeue(incidents)
	}
}

func (b *IncidentBatcher) requeue(incidents []Incident) {
	for _, inc := range incidents {
		b.queue.Enqueue(inc)
	}
}
