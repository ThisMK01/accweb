package instance

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	IncidentCollision     = "COLLISION"
	IncidentWheelSpin     = "WHEEL_SPIN"
	IncidentSpin          = "SPIN"
	IncidentLossOfControl = "LOSS_OF_CONTROL"
	IncidentTrackLimit    = "TRACK_LIMIT"
	IncidentCut           = "CUT"
	IncidentUnsafeRejoin  = "UNSAFE_REJOIN"
	IncidentDamage        = "DAMAGE"
	IncidentPenalty       = "PENALTY"
	IncidentWrongWay      = "WRONG_WAY"
)

type IncidentParticipant struct {
	SteamID   string `json:"steamId"`
	CarNumber int    `json:"carNumber"`
}

type IncidentLocation struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type IncidentClassification struct {
	Type          string             `json:"type"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
}

type TelemetryWindow struct {
	Available bool    `json:"available"`
	Start     float64 `json:"start,omitempty"`
	Impact    float64 `json:"impact,omitempty"`
	End       float64 `json:"end,omitempty"`
}

type Incident struct {
	IncidentID      string                  `json:"incidentId"`
	EventID         string                  `json:"eventId,omitempty"`
	ServerID        string                  `json:"serverId,omitempty"`
	SessionID       string                  `json:"sessionId,omitempty"`
	Timestamp       float64                 `json:"timestamp"`
	TimestampUTC    string                  `json:"timestampUtc,omitempty"`
	Lap             *int                    `json:"lap,omitempty"`
	Sector          *int                    `json:"sector,omitempty"`
	Type            string                  `json:"type"`
	Participants    []IncidentParticipant   `json:"participants"`
	Location        *IncidentLocation       `json:"location,omitempty"`
	Severity        *float64                `json:"severity,omitempty"`
	Confidence      *float64                `json:"confidence,omitempty"`
	Classification  *IncidentClassification `json:"classification,omitempty"`
	Evidence        map[string]interface{}  `json:"evidence,omitempty"`
	TelemetryWindow *TelemetryWindow        `json:"telemetryWindow,omitempty"`
	Processed       bool                    `json:"processed"`
}

type IncidentBatchPayload struct {
	EventID   string     `json:"eventId"`
	ServerID  string     `json:"serverId"`
	SessionID string     `json:"sessionId,omitempty"`
	Incidents []Incident `json:"incidents"`
}

func GenerateIncidentID(incidentType string, carNumber int) string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("inc_%d_%s_%d", time.Now().UnixNano(), incidentType, carNumber)
	}
	return fmt.Sprintf("inc_%d_%s_%d_%s", time.Now().Unix(), incidentType, carNumber, hex.EncodeToString(bytes))
}
