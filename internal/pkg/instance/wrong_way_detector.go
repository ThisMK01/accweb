package instance

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type WrongWayState int

const (
	WrongWayStateNormal WrongWayState = iota
	WrongWayStateSuspected
	WrongWayStateConfirmed
	WrongWayStateRecovering
)

type CarWrongWayTracker struct {
	State          WrongWayState
	FirstSeenTime  time.Time
	PrevSplinePos  float64
	PrevTime       time.Time
	SustainedSec   float64
	LastIncidentAt time.Time
}

type WrongWayDetectorConfig struct {
	MinSpeedKmh        float64
	SustainedThreshold time.Duration
}

func DefaultWrongWayConfig() WrongWayDetectorConfig {
	return WrongWayDetectorConfig{
		MinSpeedKmh:        22.0,                    // Must be traveling at speed, not slow repositioning
		SustainedThreshold: 2500 * time.Millisecond, // Must travel backwards for >= 2.5s
	}
}

type WrongWayDetector struct {
	collector *TelemetryCollector
	buffer    *TelemetryBuffer
	cfg       WrongWayDetectorConfig
	trackers  map[int]*CarWrongWayTracker
	lock      sync.Mutex
}

func NewWrongWayDetector(collector *TelemetryCollector, buffer *TelemetryBuffer) *WrongWayDetector {
	return NewWrongWayDetectorWithConfig(collector, buffer, DefaultWrongWayConfig())
}

func NewWrongWayDetectorWithConfig(collector *TelemetryCollector, buffer *TelemetryBuffer, cfg WrongWayDetectorConfig) *WrongWayDetector {
	return &WrongWayDetector{
		collector: collector,
		buffer:    buffer,
		cfg:       cfg,
		trackers:  make(map[int]*CarWrongWayTracker),
	}
}

func (d *WrongWayDetector) ProcessFrame(frame TelemetryFrame) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// Only evaluate on-track racing cars in forward gears (Gear >= 2 represents 1st gear and above)
	if frame.CarLocation != 1 || frame.Gear < 2 || frame.SpeedKmh < d.cfg.MinSpeedKmh {
		if tracker, ok := d.trackers[frame.CarID]; ok && tracker.State == WrongWayStateSuspected {
			tracker.State = WrongWayStateNormal
		}
		return
	}

	tracker, ok := d.trackers[frame.CarID]
	if !ok {
		tracker = &CarWrongWayTracker{
			State:         WrongWayStateNormal,
			PrevSplinePos: frame.SplinePosition,
			PrevTime:      frame.ReceivedAt,
		}
		d.trackers[frame.CarID] = tracker
		return
	}

	dSpline := frame.SplinePosition - tracker.PrevSplinePos
	tracker.PrevSplinePos = frame.SplinePosition
	tracker.PrevTime = frame.ReceivedAt

	// Ignore normal finish line wrap-around (e.g. 0.98 -> 0.02 is +0.04 forward progress)
	if dSpline < -0.5 {
		dSpline += 1.0
	} else if dSpline > 0.5 {
		dSpline -= 1.0
	}

	// Negative progression indicates traveling in reverse direction along the track
	isMovingBackwards := dSpline < -0.0005

	switch tracker.State {
	case WrongWayStateNormal:
		if isMovingBackwards {
			tracker.State = WrongWayStateSuspected
			tracker.FirstSeenTime = frame.ReceivedAt
		}

	case WrongWayStateSuspected:
		if isMovingBackwards {
			if frame.ReceivedAt.Sub(tracker.FirstSeenTime) >= d.cfg.SustainedThreshold {
				tracker.State = WrongWayStateConfirmed
				tracker.LastIncidentAt = frame.ReceivedAt
				d.emitWrongWayIncident(frame)
			}
		} else {
			// Driver turned around quickly (not sustained wrong-way)
			tracker.State = WrongWayStateNormal
		}

	case WrongWayStateConfirmed:
		if !isMovingBackwards {
			tracker.State = WrongWayStateRecovering
			tracker.FirstSeenTime = frame.ReceivedAt
		}

	case WrongWayStateRecovering:
		if !isMovingBackwards {
			// Require 2 seconds of forward driving to fully restore normal state
			if frame.ReceivedAt.Sub(tracker.FirstSeenTime) >= 2*time.Second {
				tracker.State = WrongWayStateNormal
			}
		} else {
			// Resumed wrong way driving
			tracker.State = WrongWayStateConfirmed
		}
	}
}

func (d *WrongWayDetector) emitWrongWayIncident(frame TelemetryFrame) {
	if frame.SteamID == "" {
		return
	}

	severity := 0.85
	confidence := 0.95

	incident := Incident{
		IncidentID: GenerateIncidentID(IncidentWrongWay, frame.CarNumber),
		EventID:    d.collector.eventId,
		ServerID:   d.collector.instance.GetID(),
		SessionID:  frame.SessionType,
		Timestamp:  frame.SourceTimestamp,
		Type:       IncidentWrongWay,
		Participants: []IncidentParticipant{
			{
				SteamID:   frame.SteamID,
				CarNumber: frame.CarNumber,
			},
		},
		Severity:   &severity,
		Confidence: &confidence,
		Evidence: map[string]interface{}{
			"speedKmh":       frame.SpeedKmh,
			"splinePosition": frame.SplinePosition,
			"gear":           frame.Gear,
			"lap":            frame.Lap,
		},
	}

	logrus.WithFields(logrus.Fields{
		"event_id":  d.collector.eventId,
		"carNumber": frame.CarNumber,
		"steamId":   frame.SteamID,
		"speed":     frame.SpeedKmh,
	}).Warn("Detected confirmed WRONG_WAY driving incident")

	d.collector.EnqueueIncident(incident)
}
