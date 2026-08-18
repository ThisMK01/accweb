package instance

import (
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type LOCState int

const (
	LOCStateNormal LOCState = iota
	LOCStateUnstable
	LOCStateConfirmed
	LOCStateRecovering
)

type CarLOCTracker struct {
	State              LOCState
	FirstSeenTime      time.Time
	PrevSign           int
	OscillationCount   int
	PeakYawRate        float64
	LastSignChangeTime time.Time
}

type LossOfControlConfig struct {
	MinSpeedKmh  float64
	MinThreshold float64
	Window       time.Duration
	MinReversals int
}

func DefaultLossOfControlConfig() LossOfControlConfig {
	return LossOfControlConfig{
		MinSpeedKmh:  35.0,                    // High speed instability / fish-tailing
		MinThreshold: 1.2,                     // Significant yaw velocity (~68 deg/sec)
		Window:       1500 * time.Millisecond, // Window to detect rapid oscillations
		MinReversals: 2,                       // At least 2 rapid yaw direction reversals
	}
}

type LossOfControlDetector struct {
	collector *TelemetryCollector
	buffer    *TelemetryBuffer
	cfg       LossOfControlConfig
	trackers  map[int]*CarLOCTracker
	lock      sync.Mutex
}

func NewLossOfControlDetector(collector *TelemetryCollector, buffer *TelemetryBuffer) *LossOfControlDetector {
	return NewLossOfControlDetectorWithConfig(collector, buffer, DefaultLossOfControlConfig())
}

func NewLossOfControlDetectorWithConfig(collector *TelemetryCollector, buffer *TelemetryBuffer, cfg LossOfControlConfig) *LossOfControlDetector {
	return &LossOfControlDetector{
		collector: collector,
		buffer:    buffer,
		cfg:       cfg,
		trackers:  make(map[int]*CarLOCTracker),
	}
}

func (d *LossOfControlDetector) ProcessFrame(frame TelemetryFrame) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// Precedence rule: if car is already in a full SPIN, do not double-fire LOSS_OF_CONTROL
	if d.collector != nil && d.collector.telemetryReceiver != nil {
		if d.collector.telemetryReceiver.spinDetector.IsInSpin(frame.CarID) {
			if tracker, ok := d.trackers[frame.CarID]; ok {
				tracker.State = LOCStateNormal
				tracker.OscillationCount = 0
			}
			return
		}
	}

	if frame.SpeedKmh < d.cfg.MinSpeedKmh || frame.CarLocation != 1 {
		return
	}

	tracker, ok := d.trackers[frame.CarID]
	if !ok {
		tracker = &CarLOCTracker{
			State: LOCStateNormal,
		}
		d.trackers[frame.CarID] = tracker
	}

	absYawRate := math.Abs(frame.YawRate)
	sign := 0
	if frame.YawRate > 0.4 {
		sign = 1
	} else if frame.YawRate < -0.4 {
		sign = -1
	}

	switch tracker.State {
	case LOCStateNormal:
		if absYawRate >= d.cfg.MinThreshold && sign != 0 {
			tracker.State = LOCStateUnstable
			tracker.FirstSeenTime = frame.ReceivedAt
			tracker.LastSignChangeTime = frame.ReceivedAt
			tracker.PrevSign = sign
			tracker.OscillationCount = 0
			tracker.PeakYawRate = absYawRate
		}

	case LOCStateUnstable:
		if absYawRate > tracker.PeakYawRate {
			tracker.PeakYawRate = absYawRate
		}

		// Detect directional oscillation (snapping left to right)
		if sign != 0 && sign != tracker.PrevSign {
			tracker.PrevSign = sign
			tracker.OscillationCount++
			tracker.LastSignChangeTime = frame.ReceivedAt
		}

		elapsed := frame.ReceivedAt.Sub(tracker.FirstSeenTime)

		// Trigger if >= MinReversals rapid oscillations occurred within Window duration
		if tracker.OscillationCount >= d.cfg.MinReversals && elapsed <= d.cfg.Window {
			tracker.State = LOCStateConfirmed
			d.emitLOCIncident(frame, tracker)
		} else if elapsed > d.cfg.Window {
			// Stabilized without incident
			tracker.State = LOCStateNormal
		}

	case LOCStateConfirmed:
		if absYawRate < 0.6 {
			tracker.State = LOCStateRecovering
			tracker.FirstSeenTime = frame.ReceivedAt
		}

	case LOCStateRecovering:
		if absYawRate < 0.6 {
			if frame.ReceivedAt.Sub(tracker.FirstSeenTime) >= 1500*time.Millisecond {
				tracker.State = LOCStateNormal
			}
		} else {
			tracker.FirstSeenTime = frame.ReceivedAt
		}
	}
}

func (d *LossOfControlDetector) emitLOCIncident(frame TelemetryFrame, tracker *CarLOCTracker) {
	if frame.SteamID == "" {
		return
	}

	severity := 0.60
	confidence := 0.88

	incident := Incident{
		IncidentID: GenerateIncidentID(IncidentLossOfControl, frame.CarNumber),
		EventID:    d.collector.eventId,
		ServerID:   d.collector.instance.GetID(),
		SessionID:  frame.SessionType,
		Timestamp:  frame.SourceTimestamp,
		Type:       IncidentLossOfControl,
		Participants: []IncidentParticipant{
			{
				SteamID:   frame.SteamID,
				CarNumber: frame.CarNumber,
			},
		},
		Severity:   &severity,
		Confidence: &confidence,
		Evidence: map[string]interface{}{
			"speedKmh":         frame.SpeedKmh,
			"peakYawRate":      tracker.PeakYawRate,
			"oscillations":     tracker.OscillationCount,
			"splinePosition":   frame.SplinePosition,
			"lap":              frame.Lap,
		},
		TelemetryWindow: &TelemetryWindow{
			Available: true,
			Start:     frame.SourceTimestamp - 1.5,
			Impact:    frame.SourceTimestamp,
			End:       frame.SourceTimestamp + 1.5,
		},
	}

	logrus.WithFields(logrus.Fields{
		"event_id":     d.collector.eventId,
		"carNumber":    frame.CarNumber,
		"steamId":      frame.SteamID,
		"oscillations": tracker.OscillationCount,
	}).Info("Detected vehicle LOSS_OF_CONTROL incident")

	d.collector.EnqueueIncident(incident)
}
