package instance

import (
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type SpinState int

const (
	SpinStateNormal SpinState = iota
	SpinStateCandidate
	SpinStateConfirmed
	SpinStateRecovering
)

type CarSpinTracker struct {
	State              SpinState
	StartTime          time.Time
	CandidateTime      time.Time
	PeakYawRate        float64
	CumulativeRotation float64
	PrevYaw            float64
	RecoveryTime       time.Time
}

type SpinDetectorConfig struct {
	MinSpeedKmh                float64
	CandidateYawRate           float64
	ConfirmedYawRate           float64
	CumulativeHeadingThreshold float64
}

func DefaultSpinConfig() SpinDetectorConfig {
	return SpinDetectorConfig{
		MinSpeedKmh:                20.0, // Minimum speed to be considered a dynamic race spin
		CandidateYawRate:           1.6,  // ~92 deg/sec
		ConfirmedYawRate:           2.4,  // ~137 deg/sec
		CumulativeHeadingThreshold: 1.2,  // ~70 degrees total rotation during event
	}
}

type SpinDetector struct {
	collector *TelemetryCollector
	buffer    *TelemetryBuffer
	cfg       SpinDetectorConfig
	trackers  map[int]*CarSpinTracker
	lock      sync.Mutex
}

func NewSpinDetector(collector *TelemetryCollector, buffer *TelemetryBuffer) *SpinDetector {
	return NewSpinDetectorWithConfig(collector, buffer, DefaultSpinConfig())
}

func NewSpinDetectorWithConfig(collector *TelemetryCollector, buffer *TelemetryBuffer, cfg SpinDetectorConfig) *SpinDetector {
	return &SpinDetector{
		collector: collector,
		buffer:    buffer,
		cfg:       cfg,
		trackers:  make(map[int]*CarSpinTracker),
	}
}

func (d *SpinDetector) ProcessFrame(frame TelemetryFrame) {
	d.lock.Lock()
	defer d.lock.Unlock()

	tracker, ok := d.trackers[frame.CarID]
	if !ok {
		tracker = &CarSpinTracker{
			State:   SpinStateNormal,
			PrevYaw: frame.Yaw,
		}
		d.trackers[frame.CarID] = tracker
	}

	absYawRate := math.Abs(frame.YawRate)

	// Calculate incremental heading delta
	dYaw := frame.Yaw - tracker.PrevYaw
	for dYaw > math.Pi {
		dYaw -= 2 * math.Pi
	}
	for dYaw < -math.Pi {
		dYaw += 2 * math.Pi
	}
	tracker.PrevYaw = frame.Yaw

	switch tracker.State {
	case SpinStateNormal:
		if frame.SpeedKmh >= d.cfg.MinSpeedKmh && absYawRate >= d.cfg.CandidateYawRate {
			tracker.State = SpinStateCandidate
			tracker.CandidateTime = frame.ReceivedAt
			tracker.PeakYawRate = absYawRate
			tracker.CumulativeRotation = math.Abs(dYaw)
		}

	case SpinStateCandidate:
		tracker.CumulativeRotation += math.Abs(dYaw)
		if absYawRate > tracker.PeakYawRate {
			tracker.PeakYawRate = absYawRate
		}

		elapsed := frame.ReceivedAt.Sub(tracker.CandidateTime)

		// Confirmation condition: sustained high rotation speed or large cumulative angle change
		if (absYawRate >= d.cfg.ConfirmedYawRate || tracker.CumulativeRotation >= d.cfg.CumulativeHeadingThreshold) && elapsed <= 2*time.Second {
			tracker.State = SpinStateConfirmed
			d.emitSpinIncident(frame, tracker)
		} else if elapsed > 1500*time.Millisecond && absYawRate < d.cfg.CandidateYawRate {
			// False positive (e.g. sharp chicane exit), reset to normal
			tracker.State = SpinStateNormal
		}

	case SpinStateConfirmed:
		tracker.CumulativeRotation += math.Abs(dYaw)
		if absYawRate > tracker.PeakYawRate {
			tracker.PeakYawRate = absYawRate
		}

		// Transition to recovery when car rotation slows down
		if absYawRate < 0.8 {
			tracker.State = SpinStateRecovering
			tracker.RecoveryTime = frame.ReceivedAt
		}

	case SpinStateRecovering:
		// Require stable driving for at least 1.5 seconds before resetting detector
		if absYawRate < 0.8 {
			if frame.ReceivedAt.Sub(tracker.RecoveryTime) >= 1500*time.Millisecond {
				tracker.State = SpinStateNormal
			}
		} else {
			// Still unsettled
			tracker.RecoveryTime = frame.ReceivedAt
		}
	}
}

func (d *SpinDetector) IsInSpin(carID int) bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	if t, ok := d.trackers[carID]; ok {
		return t.State == SpinStateCandidate || t.State == SpinStateConfirmed
	}
	return false
}

func (d *SpinDetector) emitSpinIncident(frame TelemetryFrame, tracker *CarSpinTracker) {
	if frame.SteamID == "" {
		return
	}

	confidence := 0.93
	severity := math.Min(1.0, math.Max(0.3, tracker.CumulativeRotation/math.Pi))

	evidence := map[string]interface{}{
		"speedKmh":           frame.SpeedKmh,
		"peakYawRate":        tracker.PeakYawRate,
		"cumulativeRotation": tracker.CumulativeRotation,
		"splinePosition":     frame.SplinePosition,
		"lap":                frame.Lap,
	}

	incident := Incident{
		IncidentID: GenerateIncidentID(IncidentSpin, frame.CarNumber),
		EventID:    d.collector.eventId,
		ServerID:   d.collector.instance.GetID(),
		SessionID:  frame.SessionType,
		Timestamp:  frame.SourceTimestamp,
		Type:       IncidentSpin,
		Participants: []IncidentParticipant{
			{
				SteamID:   frame.SteamID,
				CarNumber: frame.CarNumber,
			},
		},
		Severity:   &severity,
		Confidence: &confidence,
		Evidence:   evidence,
		TelemetryWindow: &TelemetryWindow{
			Available: true,
			Start:     frame.SourceTimestamp - 2.0,
			Impact:    frame.SourceTimestamp,
			End:       frame.SourceTimestamp + 2.0,
		},
	}

	logrus.WithFields(logrus.Fields{
		"event_id":    d.collector.eventId,
		"carNumber":   frame.CarNumber,
		"steamId":     frame.SteamID,
		"peakYawRate": tracker.PeakYawRate,
		"rotation":    tracker.CumulativeRotation,
	}).Info("Detected confirmed vehicle SPIN incident")

	d.collector.EnqueueIncident(incident)
}
