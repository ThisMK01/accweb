package instance

import "time"

// TelemetryFrame represents a normalized high-frequency vehicle telemetry snapshot
// captured locally inside ACCWeb from the ACC UDP Broadcasting stream.
type TelemetryFrame struct {
	// SourceTimestamp is the exact in-game session timestamp provided by the ACC simulation.
	SourceTimestamp float64 `json:"sourceTimestamp"`

	// ReceivedAt is the local ACCWeb system clock time when the UDP packet was received.
	ReceivedAt time.Time `json:"receivedAt"`

	// SessionTime is the elapsed session time in seconds.
	SessionTime float64 `json:"sessionTime"`

	// CarID is the ACC internal car index (e.g., 1001).
	CarID int `json:"carId"`

	// CarNumber is the driver's in-game race number (e.g., 333).
	CarNumber int `json:"carNumber"`

	// SteamID is the driver's unique persistent Steam64 ID.
	SteamID string `json:"steamId"`

	// SpeedKmh is the vehicle speed in kilometers per hour.
	SpeedKmh float64 `json:"speedKmh"`

	// Gear represents the active gear (0=R, 1=N, 2=1st, 3=2nd, etc.).
	Gear int `json:"gear"`

	// WorldPosX is the vehicle's 2D world position X on the track map.
	WorldPosX float64 `json:"worldPosX"`

	// WorldPosY is the vehicle's 2D world position Y on the track map.
	WorldPosY float64 `json:"worldPosY"`

	// Yaw is the vehicle's heading orientation in radians.
	Yaw float64 `json:"yaw"`

	// YawRate is the calculated angular velocity in radians per second (dYaw / dt).
	YawRate float64 `json:"yawRate"`

	// SplinePosition is the normalized track progression from 0.0 to 1.0 around the lap.
	SplinePosition float64 `json:"splinePosition"`

	// CarLocation indicates where the vehicle is: 0=None, 1=Track, 2=Pitlane, 3=PitEntry, 4=PitExit.
	CarLocation int `json:"carLocation"`

	// Lap is the current lap count.
	Lap int `json:"lap"`

	// Delta is the driver's delta to best session lap in milliseconds.
	Delta float64 `json:"delta"`

	// IsValidLap indicates whether the current lap is valid (no track limits exceeded).
	IsValidLap bool `json:"isValidLap"`

	// SessionType represents Practice, Qualifying, or Race.
	SessionType string `json:"sessionType"`

	// SessionPhase represents the current phase (Starting, Formation, Session, SessionOver, etc.).
	SessionPhase string `json:"sessionPhase"`
}
