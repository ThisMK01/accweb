package instance

import (
	"math"
	"testing"
	"time"
)

func TestTelemetryBufferBounded(t *testing.T) {
	buffer := NewTelemetryBuffer(2 * time.Second)

	now := time.Now()
	for i := 0; i < 20; i++ {
		buffer.Push(TelemetryFrame{
			CarID:       1001,
			CarNumber:   333,
			SteamID:     "76561198789435655",
			SpeedKmh:    150.0,
			ReceivedAt:  now.Add(time.Duration(i*100) * time.Millisecond),
			SessionTime: float64(i) * 0.1,
		})
	}

	history := buffer.GetHistory(1001)
	if len(history) == 0 {
		t.Fatalf("expected frames in history, got 0")
	}

	// Ensure all retained frames are within the 2-second retention window
	oldest := history[0]
	newest := history[len(history)-1]
	window := newest.ReceivedAt.Sub(oldest.ReceivedAt)

	if window > 2100*time.Millisecond {
		t.Errorf("expected buffer window <= 2.1s, got %v", window)
	}
}

func TestSpinDetectorSingleIncident(t *testing.T) {
	inst := &Instance{
		Path: "test_instance",
		Cfg: AccWebConfigJson{
			ID: "test_server",
			Settings: AccWebSettingsJson{
				EventID: "test_event_123",
			},
		},
	}
	collector := &TelemetryCollector{
		instance:      inst,
		eventId:       "test_event_123",
		incidentQueue: NewIncidentQueue(""),
		active:        true,
	}

	buffer := NewTelemetryBuffer(5 * time.Second)
	detector := NewSpinDetector(collector, buffer)

	now := time.Now()

	// 1. Normal driving (10 frames)
	for i := 0; i < 10; i++ {
		detector.ProcessFrame(TelemetryFrame{
			CarID:           1001,
			CarNumber:       333,
			SteamID:         "76561198789435655",
			SpeedKmh:        120.0,
			Yaw:             0.0,
			YawRate:         0.05,
			ReceivedAt:      now.Add(time.Duration(i*100) * time.Millisecond),
			SourceTimestamp: float64(i) * 0.1,
		})
	}

	if collector.incidentQueue.Len() != 0 {
		t.Fatalf("expected 0 incidents during normal driving, got %d", collector.incidentQueue.Len())
	}

	// 2. Physical Spin Sequence (25 frames of high rotation / yaw rate > 2.6 rad/s)
	yaw := 0.0
	for i := 10; i < 35; i++ {
		yaw += 0.3 // ~17 degrees per frame = rapid spin
		detector.ProcessFrame(TelemetryFrame{
			CarID:           1001,
			CarNumber:       333,
			SteamID:         "76561198789435655",
			SpeedKmh:        85.0 - float64(i-10)*2.0,
			Yaw:             yaw,
			YawRate:         2.8,
			ReceivedAt:      now.Add(time.Duration(i*100) * time.Millisecond),
			SourceTimestamp: float64(i) * 0.1,
		})
	}

	// 3. Recovery sequence (20 frames of stopped / stabilized car)
	for i := 35; i < 55; i++ {
		detector.ProcessFrame(TelemetryFrame{
			CarID:           1001,
			CarNumber:       333,
			SteamID:         "76561198789435655",
			SpeedKmh:        25.0,
			Yaw:             yaw,
			YawRate:         0.1,
			ReceivedAt:      now.Add(time.Duration(i*100) * time.Millisecond),
			SourceTimestamp: float64(i) * 0.1,
		})
	}

	// Verification: Exactly ONE SPIN Incident must be enqueued across all 55 frames
	if collector.incidentQueue.Len() != 1 {
		t.Fatalf("expected exactly 1 SPIN incident for physical spin, got %d", collector.incidentQueue.Len())
	}

	inc := collector.incidentQueue.DequeueBatch(1)[0]
	if inc.Type != IncidentSpin {
		t.Errorf("expected incident type %s, got %s", IncidentSpin, inc.Type)
	}
	if len(inc.Participants) != 1 || inc.Participants[0].SteamID != "76561198789435655" {
		t.Errorf("incorrect participant: %+v", inc.Participants)
	}
	if inc.Severity == nil || *inc.Severity < 0.3 {
		t.Errorf("expected valid severity, got %v", inc.Severity)
	}
}

func TestWrongWayDetectorSustainedThreshold(t *testing.T) {
	inst := &Instance{
		Path: "test_instance",
		Cfg: AccWebConfigJson{
			ID: "test_server",
			Settings: AccWebSettingsJson{
				EventID: "test_event_123",
			},
		},
	}
	collector := &TelemetryCollector{
		instance:      inst,
		eventId:       "test_event_123",
		incidentQueue: NewIncidentQueue(""),
		active:        true,
	}

	buffer := NewTelemetryBuffer(5 * time.Second)
	detector := NewWrongWayDetector(collector, buffer)

	now := time.Now()

	// 1. Brief reverse movement (< 1.5s): Should NOT trigger WRONG_WAY
	spline := 0.500
	for i := 0; i < 10; i++ {
		spline -= 0.001
		detector.ProcessFrame(TelemetryFrame{
			CarID:           1001,
			CarNumber:       333,
			SteamID:         "76561198789435655",
			SpeedKmh:        40.0,
			Gear:            2, // 1st gear
			CarLocation:     1, // On track
			SplinePosition:  spline,
			ReceivedAt:      now.Add(time.Duration(i*100) * time.Millisecond),
			SourceTimestamp: float64(i) * 0.1,
		})
	}

	if collector.incidentQueue.Len() != 0 {
		t.Fatalf("expected 0 incidents for brief reverse movement, got %d", collector.incidentQueue.Len())
	}

	// 2. Sustained backwards driving for >= 2.8 seconds (28 frames)
	for i := 10; i < 40; i++ {
		spline -= 0.001
		detector.ProcessFrame(TelemetryFrame{
			CarID:           1001,
			CarNumber:       333,
			SteamID:         "76561198789435655",
			SpeedKmh:        45.0,
			Gear:            2,
			CarLocation:     1,
			SplinePosition:  spline,
			ReceivedAt:      now.Add(time.Duration(i*100) * time.Millisecond),
			SourceTimestamp: float64(i) * 0.1,
		})
	}

	// Verification: Exactly ONE WRONG_WAY incident must be generated
	if collector.incidentQueue.Len() != 1 {
		t.Fatalf("expected exactly 1 WRONG_WAY incident, got %d", collector.incidentQueue.Len())
	}

	inc := collector.incidentQueue.DequeueBatch(1)[0]
	if inc.Type != IncidentWrongWay {
		t.Errorf("expected incident type %s, got %s", IncidentWrongWay, inc.Type)
	}
}

func TestLossOfControlPrecedenceWithSpin(t *testing.T) {
	inst := &Instance{
		Path: "test_instance",
		Cfg: AccWebConfigJson{
			ID: "test_server",
			Settings: AccWebSettingsJson{
				EventID: "test_event_123",
			},
		},
	}
	collector := &TelemetryCollector{
		instance:      inst,
		eventId:       "test_event_123",
		incidentQueue: NewIncidentQueue(""),
		active:        true,
	}

	buffer := NewTelemetryBuffer(5 * time.Second)
	spinDet := NewSpinDetector(collector, buffer)
	locDet := NewLossOfControlDetector(collector, buffer)

	now := time.Now()

	// Simulate fish-tailing / oscillations without full spin
	yawRate := 1.4
	for i := 0; i < 20; i++ {
		if i%4 == 0 {
			yawRate = -yawRate
		}
		locDet.ProcessFrame(TelemetryFrame{
			CarID:           1001,
			CarNumber:       333,
			SteamID:         "76561198789435655",
			SpeedKmh:        90.0,
			YawRate:         yawRate,
			CarLocation:     1,
			ReceivedAt:      now.Add(time.Duration(i*100) * time.Millisecond),
			SourceTimestamp: float64(i) * 0.1,
		})
	}

	if collector.incidentQueue.Len() != 1 {
		t.Fatalf("expected 1 LOSS_OF_CONTROL incident for fish-tailing, got %d", collector.incidentQueue.Len())
	}

	inc := collector.incidentQueue.DequeueBatch(1)[0]
	if inc.Type != IncidentLossOfControl {
		t.Errorf("expected incident type %s, got %s", IncidentLossOfControl, inc.Type)
	}
	_ = spinDet
	_ = math.Pi
}
