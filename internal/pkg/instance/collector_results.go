package instance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type ACCResultCarDriver struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	ShortName string `json:"shortName"`
	PlayerID  string `json:"playerId"`
}

type ACCResultCar struct {
	CarID       int                  `json:"carId"`
	RaceNumber  int                  `json:"raceNumber"`
	CarModel    int                  `json:"carModel"`
	CupCategory int                  `json:"cupCategory"`
	TeamName    string               `json:"teamName"`
	Drivers     []ACCResultCarDriver `json:"drivers"`
}

type ACCResultTiming struct {
	BestLap   int `json:"bestLap"`
	TotalTime int `json:"totalTime"`
	LapCount  int `json:"lapCount"`
	LastLap   int `json:"lastLap"`
}

type ACCResultLeaderboardLine struct {
	Car                       ACCResultCar       `json:"car"`
	CurrentDriver             ACCResultCarDriver `json:"currentDriver"`
	CurrentDriver_HasFinished int                `json:"currentDriver_HasFinished"`
	Timing                    ACCResultTiming    `json:"timing"`
	MissingMandatoryPitstop   int                `json:"missingMandatoryPitstop"`
}

type ACCResultLap struct {
	CarID          int   `json:"carId"`
	DriverIndex    int   `json:"driverIndex"`
	Laptime        int   `json:"laptime"`
	IsValidForBest bool  `json:"isValidForBest"`
	Splits         []int `json:"splits"`
}

type ACCResultPenalty struct {
	CarID          int    `json:"carId"`
	DriverIndex    int    `json:"driverIndex"`
	Reason         string `json:"reason"`
	Penalty        string `json:"penalty"`
	PenaltyValue   int    `json:"penaltyValue"`
	ViolationInLap int    `json:"violationInLap"`
	ClearedInLap   int    `json:"clearedInLap"`
}

type BackendPenaltyPayload struct {
	Reason         string `json:"reason"`
	Penalty        string `json:"penalty"`
	PenaltyValue   int    `json:"penaltyValue"`
	ViolationInLap int    `json:"violationInLap,omitempty"`
	ClearedInLap   int    `json:"clearedInLap,omitempty"`
}

type ACCSessionResult struct {
	BestLap          int                        `json:"bestlap"`
	Type             int                        `json:"type"`
	LeaderBoardLines []ACCResultLeaderboardLine `json:"leaderBoardLines"`
}

type ACCResultFile struct {
	SessionType      string                     `json:"sessionType"`
	TrackName        string                     `json:"trackName"`
	SessionIndex     int                        `json:"sessionIndex"`
	RaceWeekendIndex int                        `json:"raceWeekendIndex"`
	ServerName       string                     `json:"serverName"`
	LeaderBoardLines []ACCResultLeaderboardLine `json:"leaderBoardLines"`
	SessionResult    ACCSessionResult           `json:"sessionResult"`
	Laps             []ACCResultLap             `json:"laps"`
	Penalties        []ACCResultPenalty         `json:"penalties"`
}

func (r *ACCResultFile) GetLeaderBoardLines() []ACCResultLeaderboardLine {
	if len(r.SessionResult.LeaderBoardLines) > 0 {
		return r.SessionResult.LeaderBoardLines
	}
	return r.LeaderBoardLines
}

type BackendDriverResultPayload struct {
	SteamID          string                  `json:"steamId"`
	CarNumber        int                     `json:"carNumber"`
	StartPosition    *int                    `json:"startPosition,omitempty"`
	FinishPosition   *int                    `json:"finishPosition,omitempty"`
	Status           string                  `json:"status"`
	BestLapTime      *int                    `json:"bestLapTime,omitempty"`
	TotalTime        *int                    `json:"totalTime,omitempty"`
	LapCount         int                     `json:"lapCount"`
	CutsCount        int                     `json:"cutsCount"`
	InvalidLapsCount int                     `json:"invalidLapsCount"`
	Penalties        []BackendPenaltyPayload `json:"penalties"`
}

type BackendRaceResultPayload struct {
	ServerID string                       `json:"serverId"`
	Results  []BackendDriverResultPayload `json:"results"`
}

func CleanSteamID(rawId string) string {
	cleaned := strings.TrimSpace(rawId)
	if strings.HasPrefix(cleaned, "S") || strings.HasPrefix(cleaned, "s") {
		cleaned = cleaned[1:]
	}
	return cleaned
}

func (c *TelemetryCollector) watchResults() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	resultsDir := filepath.Join(c.instance.Path, "results")

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.checkNewResultFiles(resultsDir)
		}
	}
}

func (c *TelemetryCollector) checkNewResultFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filename := entry.Name()
		c.lock.Lock()
		alreadyProcessed := c.processedFiles[filename]
		c.lock.Unlock()

		if alreadyProcessed {
			continue
		}

		filePath := filepath.Join(dir, filename)
		if err := c.processResultFile(filePath, filename); err == nil {
			c.lock.Lock()
			c.processedFiles[filename] = true
			c.lock.Unlock()
		} else {
			logrus.WithError(err).WithField("file", filename).Warn("Failed to process ACC result file; will retry")
		}
	}
}

func (c *TelemetryCollector) processResultFile(filePath, filename string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Strip UTF-8 BOM if present
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	// Decode UTF-16LE (with BOM 0xFF 0xFE or raw UTF-16LE without BOM)
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		decoded, _, err := transform.Bytes(decoder, data[2:])
		if err != nil {
			return fmt.Errorf("utf-16 decode error: %w", err)
		}
		data = decoded
	} else if len(data) >= 2 && (data[1] == 0x00 || bytes.IndexByte(data, 0x00) != -1) {
		decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
		decoded, _, err := transform.Bytes(decoder, data)
		if err != nil {
			return fmt.Errorf("utf-16 decode error: %w", err)
		}
		data = decoded
	}

	data = bytes.TrimSpace(data)

	// Make sure file is completely written by attempting unmarshal
	var result ACCResultFile
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	sessionType := strings.ToUpper(strings.TrimSpace(result.SessionType))
	if sessionType == "" {
		if strings.HasSuffix(strings.ToUpper(filename), "_Q.JSON") {
			sessionType = "Q"
		} else if strings.HasSuffix(strings.ToUpper(filename), "_R.JSON") {
			sessionType = "R"
		}
	}

	lines := result.GetLeaderBoardLines()

	logrus.WithFields(logrus.Fields{
		"server_id":    c.instance.GetID(),
		"event_id":     c.eventId,
		"session_type": sessionType,
		"file":         filename,
		"drivers":      len(lines),
	}).Info("Processing ACC session result file")

	if sessionType == "Q" {
		return c.processQualifyingResult(&result)
	} else if sessionType == "R" {
		return c.processRaceResult(&result)
	}

	// FP or other sessions: mark as processed without syncing
	return nil
}

func (c *TelemetryCollector) processQualifyingResult(res *ACCResultFile) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	lines := res.GetLeaderBoardLines()

	for i, line := range lines {
		steamId := ""
		if len(line.Car.Drivers) > 0 {
			steamId = CleanSteamID(line.Car.Drivers[0].PlayerID)
		}
		if steamId == "" && line.CurrentDriver.PlayerID != "" {
			steamId = CleanSteamID(line.CurrentDriver.PlayerID)
		}

		startPos := i + 1
		if steamId != "" {
			c.qualifyingGrid[steamId] = startPos
			c.qualifyingGrid[fmt.Sprintf("%s_%d", steamId, line.Car.RaceNumber)] = startPos
		}
		if line.Car.RaceNumber > 0 {
			c.qualifyingGrid[fmt.Sprintf("car_%d", line.Car.RaceNumber)] = startPos
		}
	}

	logrus.WithFields(logrus.Fields{
		"event_id": c.eventId,
		"drivers":  len(lines),
	}).Info("Cached Qualifying starting grid in memory. Awaiting Race completion to send final payload.")

	return nil
}

func (c *TelemetryCollector) processRaceResult(res *ACCResultFile) error {
	var driverResults []BackendDriverResultPayload

	c.lock.Lock()
	grid := make(map[string]int)
	for k, v := range c.qualifyingGrid {
		grid[k] = v
	}
	c.lock.Unlock()

	// Map penalties by carId
	penaltiesByCar := make(map[int][]BackendPenaltyPayload)
	for _, p := range res.Penalties {
		penaltiesByCar[p.CarID] = append(penaltiesByCar[p.CarID], BackendPenaltyPayload{
			Reason:         p.Reason,
			Penalty:        p.Penalty,
			PenaltyValue:   p.PenaltyValue,
			ViolationInLap: p.ViolationInLap,
			ClearedInLap:   p.ClearedInLap,
		})

		// Also enqueue as a normalized PENALTY Incident
		driver := c.resolveDriver(p.CarID)
		if driver.SteamID != "" {
			c.EnqueueIncident(Incident{
				IncidentID: GenerateIncidentID(IncidentPenalty, driver.CarNumber),
				EventID:    c.eventId,
				ServerID:   c.instance.GetID(),
				SessionID:  res.SessionType,
				Timestamp:  float64(time.Now().Unix()),
				Lap:        &p.ViolationInLap,
				Type:       IncidentPenalty,
				Participants: []IncidentParticipant{
					{
						SteamID:   driver.SteamID,
						CarNumber: driver.CarNumber,
					},
				},
				Evidence: map[string]interface{}{
					"reason":         p.Reason,
					"penalty":        p.Penalty,
					"penaltyValue":   p.PenaltyValue,
					"violationInLap": p.ViolationInLap,
					"clearedInLap":   p.ClearedInLap,
				},
			})
		}
	}

	// Map invalid laps by carId
	invalidLapsByCar := make(map[int]int)
	for _, lap := range res.Laps {
		if !lap.IsValidForBest {
			invalidLapsByCar[lap.CarID]++
		}
	}

	lines := res.GetLeaderBoardLines()

	for i, line := range lines {
		steamId := ""
		if len(line.Car.Drivers) > 0 {
			steamId = CleanSteamID(line.Car.Drivers[0].PlayerID)
		}
		if steamId == "" && line.CurrentDriver.PlayerID != "" {
			steamId = CleanSteamID(line.CurrentDriver.PlayerID)
		}

		if steamId == "" {
			continue
		}

		var startPos *int
		if pos, ok := grid[fmt.Sprintf("%s_%d", steamId, line.Car.RaceNumber)]; ok {
			p := pos
			startPos = &p
		} else if pos, ok := grid[steamId]; ok {
			p := pos
			startPos = &p
		} else if pos, ok := grid[fmt.Sprintf("car_%d", line.Car.RaceNumber)]; ok {
			p := pos
			startPos = &p
		}

		finishPos := i + 1
		status := "DNS"
		if line.CurrentDriver_HasFinished == 1 || line.Timing.LapCount > 0 {
			status = "FINISHED"
		}

		var bestLap *int
		if line.Timing.BestLap > 0 && line.Timing.BestLap < 2147483640 {
			val := line.Timing.BestLap
			bestLap = &val
		}

		var totalTime *int
		if line.Timing.TotalTime > 0 {
			val := line.Timing.TotalTime
			totalTime = &val
		}

		cuts := c.GetCarCuts(line.Car.CarID, line.Car.RaceNumber)
		invalidLaps := invalidLapsByCar[line.Car.CarID]
		if cuts < invalidLaps {
			cuts = invalidLaps
		}

		driverPenalties := penaltiesByCar[line.Car.CarID]
		if driverPenalties == nil {
			driverPenalties = []BackendPenaltyPayload{}
		}

		driverResults = append(driverResults, BackendDriverResultPayload{
			SteamID:          steamId,
			CarNumber:        line.Car.RaceNumber,
			StartPosition:    startPos,
			FinishPosition:   &finishPos,
			Status:           status,
			BestLapTime:      bestLap,
			TotalTime:        totalTime,
			LapCount:         line.Timing.LapCount,
			CutsCount:        cuts,
			InvalidLapsCount: invalidLaps,
			Penalties:        driverPenalties,
		})
	}

	if len(driverResults) == 0 {
		return nil
	}

	payload := BackendRaceResultPayload{
		ServerID: c.instance.GetID(),
		Results:  driverResults,
	}

	path := fmt.Sprintf("/race-results/%s", c.eventId)
	if err := c.sendHttpRequest("POST", path, payload); err != nil {
		return fmt.Errorf("sync race results error: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"event_id": c.eventId,
		"drivers":  len(driverResults),
	}).Info("Successfully synced final Race results to backend")

	// Flush all accumulated incidents after the race finishes
	if err := c.FlushIncidentsToBackend(); err != nil {
		logrus.WithError(err).Warn("Failed to flush incidents after race results sync")
	}

	return nil
}

func (c *TelemetryCollector) FlushIncidentsToBackend() error {
	incidents := c.incidentQueue.Drain()
	if len(incidents) == 0 {
		logrus.Info("No incidents to flush after the race")
		return nil
	}

	payload := IncidentBatchPayload{
		EventID:   c.eventId,
		ServerID:  c.instance.GetID(),
		SessionID: "live_session",
		Incidents: incidents,
	}

	path := fmt.Sprintf("/events/%s/incidents/batch", c.eventId)
	if err := c.sendHttpRequest("POST", path, payload); err != nil {
		// Put them back in the queue if it failed so we can retry
		c.lock.Lock()
		c.incidentQueue.items = append(incidents, c.incidentQueue.items...)
		c.lock.Unlock()
		c.incidentQueue.SaveToDisk()
		return fmt.Errorf("failed to sync incident batch to backend: %w", err)
	}

	// Success: Delete the persistent file
	if c.incidentQueue.persistPath != "" {
		_ = os.Remove(c.incidentQueue.persistPath)
	}
	logrus.WithField("count", len(incidents)).Info("Successfully flushed all queued incidents to backend after the race")
	return nil
}
