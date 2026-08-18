package instance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type TelemetryCollector struct {
	instance          *Instance
	backendApiUrl     string
	eventId           string
	stopChan          chan struct{}
	processedFiles    map[string]bool
	qualifyingGrid    map[string]int
	liveCutsCount     map[int]int
	incidentQueue     *IncidentQueue
	incidentBatcher   *IncidentBatcher
	telemetryBuffer   *TelemetryBuffer
	telemetryReceiver *TelemetryReceiver
	active            bool
	lock              sync.Mutex
}

func NewTelemetryCollector(inst *Instance, backendUrl string) *TelemetryCollector {
	apiUrl := "http://localhost:5000"
	if backendUrl != "" {
		apiUrl = strings.TrimRight(backendUrl, "/")
	}

	c := &TelemetryCollector{
		instance:        inst,
		backendApiUrl:   apiUrl,
		eventId:         inst.Cfg.Settings.EventID,
		processedFiles:  make(map[string]bool),
		qualifyingGrid:  make(map[string]int),
		liveCutsCount:   make(map[int]int),
		incidentQueue:   NewIncidentQueue(inst.Path),
		telemetryBuffer: NewTelemetryBuffer(5 * time.Second),
		stopChan:        make(chan struct{}),
	}
	c.incidentBatcher = NewIncidentBatcher(c, c.incidentQueue)
	c.telemetryReceiver = NewTelemetryReceiver(c, c.telemetryBuffer)
	return c
}

func (c *TelemetryCollector) EnqueueIncident(inc Incident) {
	if !c.IsActive() {
		return
	}
	if inc.EventID == "" {
		inc.EventID = c.eventId
	}
	if inc.ServerID == "" {
		inc.ServerID = c.instance.GetID()
	}
	c.incidentQueue.Enqueue(inc)
}

func (c *TelemetryCollector) RecordCut(carId int) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.liveCutsCount[carId]++
}

func (c *TelemetryCollector) GetCarCuts(carId, raceNumber int) int {
	c.lock.Lock()
	defer c.lock.Unlock()
	if count, ok := c.liveCutsCount[carId]; ok && count > 0 {
		return count
	}
	if count, ok := c.liveCutsCount[raceNumber]; ok && count > 0 {
		return count
	}
	return 0
}

func (c *TelemetryCollector) Start() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.active {
		return
	}

	if !c.instance.Cfg.Settings.CollectorEnabled || strings.TrimSpace(c.instance.Cfg.Settings.EventID) == "" {
		logrus.WithField("server_id", c.instance.GetID()).Debug("Collector not enabled or event_id empty; skipping collector start")
		return
	}

	c.eventId = strings.TrimSpace(c.instance.Cfg.Settings.EventID)
	c.active = true
	c.stopChan = make(chan struct{})

	// Load any persistent queue items from disk and start batcher
	c.incidentQueue.LoadFromDisk()

	// Mark all existing result files so we only process new ones from this run
	resultsDir := filepath.Join(c.instance.Path, "results")
	if entries, err := os.ReadDir(resultsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				c.processedFiles[entry.Name()] = true
			}
		}
	}

	logrus.WithFields(logrus.Fields{
		"server_id": c.instance.GetID(),
		"event_id":  c.eventId,
	}).Info("Telemetry Collector started")

	// Start incident batcher
	c.incidentBatcher.Start()

	// Start UDP Telemetry Receiver on broadcasting port (default 9000 or udpPort-600)
	bPort := 9000
	if c.instance.AccCfg.Configuration.UdpPort > 1000 {
		bPort = c.instance.AccCfg.Configuration.UdpPort - 600
	}
	_ = c.telemetryReceiver.Start(bPort, "asd")

	go c.watchResults()
}

func (c *TelemetryCollector) Stop() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.active {
		return
	}

	c.active = false
	close(c.stopChan)

	if c.telemetryReceiver != nil {
		c.telemetryReceiver.Stop()
	}

	if c.incidentBatcher != nil {
		c.incidentBatcher.Stop()
	}

	// Persist any un-flushed incidents to disk
	c.incidentQueue.SaveToDisk()

	logrus.WithField("server_id", c.instance.GetID()).Info("Telemetry Collector stopped")
}

func (c *TelemetryCollector) IsActive() bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.active
}

func (c *TelemetryCollector) sendHttpRequest(method, path string, payload interface{}) error {
	endpoint := fmt.Sprintf("%s%s", c.backendApiUrl, path)

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload error: %w", err)
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("create http request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request error to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("backend returned status %d from %s", resp.StatusCode, endpoint)
	}

	return nil
}
