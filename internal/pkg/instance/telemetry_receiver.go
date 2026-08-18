package instance

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type DriverMapping struct {
	CarNumber int
	SteamID   string
}

type TelemetryReceiver struct {
	collector       *TelemetryCollector
	conn            *net.UDPConn
	serverAddr      *net.UDPAddr
	buffer          *TelemetryBuffer
	driverMap       map[int]DriverMapping
	prevYawByCar    map[int]float64
	prevTimeByCar   map[int]time.Time
	spinDetector    *SpinDetector
	locDetector     *LossOfControlDetector
	wrongWayDetect  *WrongWayDetector
	sessionType     string
	sessionPhase    string
	sessionTime     float64
	stopChan        chan struct{}
	running         bool
	lock            sync.RWMutex
}

func NewTelemetryReceiver(collector *TelemetryCollector, buffer *TelemetryBuffer) *TelemetryReceiver {
	r := &TelemetryReceiver{
		collector:     collector,
		buffer:        buffer,
		driverMap:     make(map[int]DriverMapping),
		prevYawByCar:  make(map[int]float64),
		prevTimeByCar: make(map[int]time.Time),
		stopChan:      make(chan struct{}),
	}
	r.spinDetector = NewSpinDetector(collector, buffer)
	r.locDetector = NewLossOfControlDetector(collector, buffer)
	r.wrongWayDetect = NewWrongWayDetector(collector, buffer)
	return r
}

func (r *TelemetryReceiver) Start(port int, password string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.running {
		return nil
	}

	if port <= 0 {
		port = 9000
	}

	serverAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("resolve UDP addr error: %w", err)
	}

	conn, err := net.DialUDP("udp4", nil, serverAddr)
	if err != nil {
		return fmt.Errorf("dial UDP error: %w", err)
	}

	r.conn = conn
	r.serverAddr = serverAddr
	r.running = true
	r.stopChan = make(chan struct{})

	// Send Registration Request packet (Protocol V4, 100ms update interval)
	r.sendRegistrationRequest(password)

	go r.listenLoop()
	go r.heartbeatLoop(password)

	logrus.WithFields(logrus.Fields{
		"broadcasting_port": port,
		"server_id":         r.collector.instance.GetID(),
	}).Info("ACC UDP Telemetry Receiver connected and listening")

	return nil
}

func (r *TelemetryReceiver) Stop() {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.running {
		return
	}

	r.running = false
	close(r.stopChan)

	if r.conn != nil {
		_ = r.conn.Close()
	}

	logrus.WithField("server_id", r.collector.instance.GetID()).Info("ACC UDP Telemetry Receiver stopped")
}

func (r *TelemetryReceiver) sendRegistrationRequest(password string) {
	buf := new(bytes.Buffer)
	buf.WriteByte(1) // MessageType: RegistrationRequest
	buf.WriteByte(4) // ProtocolVersion: 4

	writeString(buf, "accweb_collector")
	writeString(buf, password)
	_ = binary.Write(buf, binary.LittleEndian, int32(100)) // 100ms interval (10Hz)
	writeString(buf, "")                                   // Command password

	if r.conn != nil {
		_, _ = r.conn.Write(buf.Bytes())
	}
}

func (r *TelemetryReceiver) heartbeatLoop(password string) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.lock.RLock()
			running := r.running
			r.lock.RUnlock()
			if running {
				r.sendRegistrationRequest(password)
			}
		}
	}
}

func (r *TelemetryReceiver) listenLoop() {
	buf := make([]byte, 4096)

	for {
		select {
		case <-r.stopChan:
			return
		default:
		}

		if r.conn == nil {
			return
		}

		n, err := r.conn.Read(buf)
		if err != nil {
			r.lock.RLock()
			running := r.running
			r.lock.RUnlock()
			if !running {
				return
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}

		if n > 0 {
			r.processPacket(buf[:n])
		}
	}
}

func (r *TelemetryReceiver) processPacket(data []byte) {
	reader := bytes.NewReader(data)
	msgType, err := reader.ReadByte()
	if err != nil {
		return
	}

	switch msgType {
	case 2: // REALTIME_UPDATE
		r.handleRealtimeUpdate(reader)
	case 3: // REALTIME_CAR_UPDATE
		r.handleRealtimeCarUpdate(reader)
	case 4: // ENTRY_LIST
		r.handleEntryList(reader)
	}
}

func (r *TelemetryReceiver) handleRealtimeUpdate(reader *bytes.Reader) {
	_, _ = readInt32(reader) // eventIndex
	_, _ = readInt32(reader) // sessionIndex

	sTyp, _ := reader.ReadByte()
	sessionTypes := map[byte]string{0: "Practice", 4: "Qualifying", 10: "Race"}
	if name, ok := sessionTypes[sTyp]; ok {
		r.sessionType = name
	} else {
		r.sessionType = "Session"
	}

	phase, _ := reader.ReadByte()
	phases := map[byte]string{0: "None", 1: "Starting", 2: "PreFormation", 3: "Formation", 4: "Session", 5: "SessionOver", 6: "PostSession"}
	if pName, ok := phases[phase]; ok {
		r.sessionPhase = pName
	}

	sTime, _ := readFloat32(reader)
	r.sessionTime = float64(sTime)
}

func (r *TelemetryReceiver) handleEntryList(reader *bytes.Reader) {
	_, _ = readInt32(reader) // connectionId
	carCount, err := readInt16(reader)
	if err != nil {
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	for i := 0; i < int(carCount); i++ {
		carIndex, _ := readInt16(reader)
		_, _ = reader.ReadByte() // modelType
		_, _ = readString(reader) // teamName
		raceNumber, _ := readInt32(reader)
		_, _ = reader.ReadByte() // cupCategory
		currentDriverIdx, _ := reader.ReadByte()
		_, _ = reader.ReadByte() // nationality

		driverCount, _ := reader.ReadByte()
		var steamId string
		for d := 0; d < int(driverCount); d++ {
			_, _ = readString(reader) // firstName
			_, _ = readString(reader) // lastName
			_, _ = readString(reader) // shortName
			_, _ = reader.ReadByte()  // category
			pId, _ := readString(reader)
			if byte(d) == currentDriverIdx || steamId == "" {
				steamId = CleanSteamID(pId)
			}
		}

		r.driverMap[int(carIndex)] = DriverMapping{
			CarNumber: int(raceNumber),
			SteamID:   steamId,
		}
	}
}

func (r *TelemetryReceiver) handleRealtimeCarUpdate(reader *bytes.Reader) {
	carIndex, err := readInt16(reader)
	if err != nil {
		return
	}
	_, _ = readInt16(reader) // driverIndex
	_, _ = reader.ReadByte() // driverCount
	gear, _ := reader.ReadByte()
	worldPosX, _ := readFloat32(reader)
	worldPosY, _ := readFloat32(reader)
	yaw, _ := readFloat32(reader)
	carLocation, _ := reader.ReadByte()
	kmh, _ := readInt16(reader)
	_, _ = reader.ReadByte() // position
	_, _ = reader.ReadByte() // trackPosition
	splinePos, _ := readFloat32(reader)
	delta, _ := readFloat32(reader)

	now := time.Now()
	cId := int(carIndex)

	// Resolve Driver Identity
	r.lock.RLock()
	dInfo := r.driverMap[cId]
	prevYaw, hasPrevYaw := r.prevYawByCar[cId]
	prevTime, hasPrevTime := r.prevTimeByCar[cId]
	r.lock.RUnlock()

	steamId := dInfo.SteamID
	carNum := dInfo.CarNumber

	// Fallback to LiveState resolution if not in EntryList packet yet
	if steamId == "" && r.collector.instance.Live != nil {
		if car := r.collector.instance.Live.GetCar(cId); car != nil {
			carNum = car.RaceNumber
			if len(car.Drivers) > 0 && car.Drivers[0] != nil {
				steamId = CleanSteamID(car.Drivers[0].PlayerID)
			}
		}
	}

	// Calculate angular velocity (YawRate)
	yawRate := 0.0
	currentYaw := float64(yaw)
	if hasPrevYaw && hasPrevTime {
		dt := now.Sub(prevTime).Seconds()
		if dt > 0.01 && dt < 1.0 {
			dYaw := currentYaw - prevYaw
			// Normalize dYaw to [-pi, pi]
			for dYaw > math.Pi {
				dYaw -= 2 * math.Pi
			}
			for dYaw < -math.Pi {
				dYaw += 2 * math.Pi
			}
			yawRate = dYaw / dt
		}
	}

	r.lock.Lock()
	r.prevYawByCar[cId] = currentYaw
	r.prevTimeByCar[cId] = now
	r.lock.Unlock()

	frame := TelemetryFrame{
		SourceTimestamp: r.sessionTime,
		ReceivedAt:      now,
		SessionTime:     r.sessionTime,
		CarID:           cId,
		CarNumber:       carNum,
		SteamID:         steamId,
		SpeedKmh:        float64(kmh),
		Gear:            int(gear),
		WorldPosX:       float64(worldPosX),
		WorldPosY:       float64(worldPosY),
		Yaw:             currentYaw,
		YawRate:         yawRate,
		SplinePosition:  float64(splinePos),
		CarLocation:     int(carLocation),
		Delta:           float64(delta),
		SessionType:     r.sessionType,
		SessionPhase:    r.sessionPhase,
	}

	// 1. Push to Bounded Local Telemetry Buffer
	r.buffer.Push(frame)

	// 2. Feed to Stateful Detectors
	r.spinDetector.ProcessFrame(frame)
	r.locDetector.ProcessFrame(frame)
	r.wrongWayDetect.ProcessFrame(frame)
}

// Binary stream helpers
func readByte(r *bytes.Reader) (byte, error) {
	return r.ReadByte()
}

func readInt16(r *bytes.Reader) (int16, error) {
	var val int16
	err := binary.Read(r, binary.LittleEndian, &val)
	return val, err
}

func readInt32(r *bytes.Reader) (int32, error) {
	var val int32
	err := binary.Read(r, binary.LittleEndian, &val)
	return val, err
}

func readFloat32(r *bytes.Reader) (float32, error) {
	var val float32
	err := binary.Read(r, binary.LittleEndian, &val)
	return val, err
}

func readString(r *bytes.Reader) (string, error) {
	var length uint16
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return "", err
	}
	if length == 0 {
		return "", nil
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func writeString(buf *bytes.Buffer, s string) {
	b := []byte(s)
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(b)))
	buf.Write(b)
}
