package instance

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	// Matches collision patterns from ACC dedicated server logs
	collisionPattern1 = regexp.MustCompile(`(?i)Car contact between carId (\d+) and carId (\d+)`)
	collisionPattern2 = regexp.MustCompile(`(?i)Car (\d+) contact with Car (\d+)`)
	collisionPattern3 = regexp.MustCompile(`(?i)collision(?: event)?:? car (\d+) (?:and|with) car (\d+)(?: at ([\d.]+))?`)

	// Matches track limit cut patterns from ACC dedicated server logs
	trackCutPattern = regexp.MustCompile(`(?i)carId (\d+).*?(?:HasCut|hasCut)`)
)

type BackendCollisionUser struct {
	CarNumber int    `json:"carNumber"`
	SteamID   string `json:"steamId"`
}

type BackendCollisionBtw struct {
	User1 BackendCollisionUser `json:"user1"`
	User2 BackendCollisionUser `json:"user2"`
}

type BackendCollisionPayload struct {
	ServerID        string              `json:"serverId"`
	TimeOfCollision float64             `json:"timeOfCollision"`
	CollisionBtw    BackendCollisionBtw `json:"collisionBtw"`
}

func (c *TelemetryCollector) HandleLogLine(line string) {
	if !c.IsActive() {
		return
	}

	cleanLine := strings.TrimSpace(line)
	if cleanLine == "" {
		return
	}

	// Check for track cut / track limit violations
	if strings.Contains(cleanLine, "HasCut") || strings.Contains(cleanLine, "hasCut") {
		if match := trackCutPattern.FindStringSubmatch(cleanLine); len(match) >= 2 {
			if id, err := strconv.Atoi(match[1]); err == nil {
				c.RecordCut(id)

				// Also enqueue a normalized Track Limit / Cut Incident
				driver := c.resolveDriver(id)
				if driver.SteamID != "" {
					severity := 0.25
					c.EnqueueIncident(Incident{
						IncidentID: GenerateIncidentID(IncidentTrackLimit, driver.CarNumber),
						EventID:    c.eventId,
						ServerID:   c.instance.GetID(),
						SessionID:  "live_session",
						Timestamp:  float64(time.Now().Unix()),
						Type:       IncidentTrackLimit,
						Participants: []IncidentParticipant{
							{
								SteamID:   driver.SteamID,
								CarNumber: driver.CarNumber,
							},
						},
						Severity: &severity,
						Evidence: map[string]interface{}{
							"rawLog": cleanLine,
						},
					})
				}
			}
		}
	}

	// Try matching collision regex patterns
	var carId1, carId2 int
	var timeSec float64 = 0.0
	matched := false

	if match := collisionPattern1.FindStringSubmatch(cleanLine); len(match) >= 3 {
		id1, _ := strconv.Atoi(match[1])
		id2, _ := strconv.Atoi(match[2])
		carId1, carId2 = id1, id2
		matched = true
	} else if match := collisionPattern2.FindStringSubmatch(cleanLine); len(match) >= 3 {
		id1, _ := strconv.Atoi(match[1])
		id2, _ := strconv.Atoi(match[2])
		carId1, carId2 = id1, id2
		matched = true
	} else if match := collisionPattern3.FindStringSubmatch(cleanLine); len(match) >= 3 {
		id1, _ := strconv.Atoi(match[1])
		id2, _ := strconv.Atoi(match[2])
		carId1, carId2 = id1, id2
		if len(match) >= 4 && match[3] != "" {
			t, _ := strconv.ParseFloat(match[3], 64)
			timeSec = t
		}
		matched = true
	}

	if !matched {
		return
	}

	// Fallback to relative session elapsed time if not present in log
	if timeSec <= 0 {
		timeSec = float64(time.Now().Unix())
	}

	c.RecordCollision(carId1, carId2, timeSec)
}

func (c *TelemetryCollector) RecordCollision(carId1, carId2 int, timeOfCollision float64) {
	if !c.IsActive() {
		return
	}

	user1 := c.resolveDriver(carId1)
	user2 := c.resolveDriver(carId2)

	if user1.SteamID == "" && user2.SteamID == "" {
		logrus.WithFields(logrus.Fields{
			"carId1": carId1,
			"carId2": carId2,
		}).Debug("Collision detected but could not resolve steam IDs; will still record with available info")
	}

	if user1.SteamID == "" {
		user1.SteamID = fmt.Sprintf("UNKNOWN_%d", carId1)
	}
	if user2.SteamID == "" {
		user2.SteamID = fmt.Sprintf("UNKNOWN_%d", carId2)
	}

	// 1. Enqueue normalized Incident into IncidentQueue for asynchronous batching
	severity := 0.65
	c.EnqueueIncident(Incident{
		IncidentID: GenerateIncidentID(IncidentCollision, user1.CarNumber),
		EventID:    c.eventId,
		ServerID:   c.instance.GetID(),
		SessionID:  "live_session",
		Timestamp:  timeOfCollision,
		Type:       IncidentCollision,
		Participants: []IncidentParticipant{
			{
				SteamID:   user1.SteamID,
				CarNumber: user1.CarNumber,
			},
			{
				SteamID:   user2.SteamID,
				CarNumber: user2.CarNumber,
			},
		},
		Severity: &severity,
		Evidence: map[string]interface{}{
			"timeOfCollision": timeOfCollision,
			"carId1":          carId1,
			"carId2":          carId2,
		},
	})

	// 2. Also send to legacy collisions endpoint for full backward compatibility
	payload := BackendCollisionPayload{
		ServerID:        c.instance.GetID(),
		TimeOfCollision: timeOfCollision,
		CollisionBtw: BackendCollisionBtw{
			User1: user1,
			User2: user2,
		},
	}

	path := fmt.Sprintf("/collisions/%s", c.eventId)
	go func() {
		if err := c.sendHttpRequest("POST", path, payload); err != nil {
			logrus.WithError(err).WithField("event_id", c.eventId).Debug("Legacy collision POST returned error or deprecated")
		}
	}()
}

func (c *TelemetryCollector) resolveDriver(carId int) BackendCollisionUser {
	res := BackendCollisionUser{
		CarNumber: carId,
		SteamID:   "",
	}

	if c.instance.Live != nil {
		car := c.instance.Live.GetCar(carId)
		if car != nil {
			if car.RaceNumber > 0 {
				res.CarNumber = car.RaceNumber
			}
			if len(car.Drivers) > 0 && car.Drivers[0] != nil {
				res.SteamID = CleanSteamID(car.Drivers[0].PlayerID)
			}
		}
	}

	return res
}
