package instance

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

	"github.com/assetto-corsa-web/accweb/internal/pkg/cfg"
	"github.com/assetto-corsa-web/accweb/internal/pkg/event"
	"github.com/assetto-corsa-web/accweb/internal/pkg/helper"
)

const (
	accDedicatedServerFile = "accServer.exe"
	accCfgDir              = "cfg"
	accServerLogDir        = "log"
	accServerLogFile       = "server.log"
)

var (
	ErrServerCantBeRunning = errors.New("server instance cant be running to perform this action")
	ErrServerDirIsInvalid  = errors.New("server directory is invalid")
	ErrInvalidCoreAffinity = errors.New("invalid core affinity value")
	ErrInvalidCpuPriority  = errors.New("invalid cpu priority value")

	outputFiltersEr = []*regexp.Regexp{
		regexp.MustCompile(`^==ERR: onCarUpdate \(\d+?\): timestamp is \d+? ms in the future`),
		regexp.MustCompile(`^Server was running late for \d step\(s\), not enough CPU power`),
		regexp.MustCompile(`^Udp message count \(\d+? clients\)`),
		regexp.MustCompile(`^Tcp message count \(\d+? clients\)`),
	}
)

type Instance struct {
	Path      string
	Cfg       AccWebConfigJson
	AccCfg    AccConfigFiles
	Live      *LiveState
	Collector *TelemetryCollector

	cmd    *exec.Cmd
	cmdOut io.ReadCloser

	lock sync.Mutex
}

func (s *Instance) GetID() string {
	return s.Cfg.ID
}

func (s *Instance) Start() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.IsRunning() {
		return ErrServerCantBeRunning
	}

	if err := s.prepareInstanceDir(); err != nil {
		return err
	}

	s.prepareCommandAndArgs()

	if err := s.prepareCmdLogHandler(); err != nil {
		return err
	}

	event.EmmitEventInstanceBeforeStart(s.ToEIB())

	if err := s.cmd.Start(); err != nil {
		return err
	}

	s.Live.SetServerState(ServerStateStarting)

	logrus.WithField("server_id", s.GetID()).WithField("pid", s.GetProcessID()).Info("acc server started")

	// Activate Telemetry Collector if enabled
	if s.Collector == nil {
		s.Collector = NewTelemetryCollector(s, cfg.BackendApiUrl())
	}
	s.Collector.Start()

	event.EmmitEventInstanceStarted(s.ToEIB())

	go s.wait()

	return nil
}

func (s *Instance) Stop() error {
	if !s.IsRunning() {
		return nil
	}

	s.Live.SetServerState(ServerStateStoping)

	// Stop Telemetry Collector
	if s.Collector != nil {
		s.Collector.Stop()
	}

	event.EmmitEventInstanceBeforeStop(s.ToEIB())

	if runtime.GOOS == "linux" && !cfg.SkipWine() {
		// Kill the spawned wine command
		_ = s.cmd.Process.Kill()

		// Find and kill the orphaned accServer.exe process belonging to this instance
		if files, err := os.ReadDir("/proc"); err == nil {
			for _, f := range files {
				if f.IsDir() {
					pidStr := f.Name()
					if pid, err := strconv.Atoi(pidStr); err == nil {
						if exePath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil && strings.HasSuffix(exePath, "accServer.exe") {
							if cwdPath, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); err == nil && filepath.Clean(cwdPath) == filepath.Clean(s.Path) {
								if p, err := os.FindProcess(pid); err == nil {
									_ = p.Kill()
									logrus.WithField("server_id", s.GetID()).Infof("Killed orphaned accServer.exe process with PID %d", pid)
								}
							}
						}
					}
				}
			}
		}
	} else {
		if err := s.cmd.Process.Kill(); err != nil {
			logrus.WithField("server_id", s.GetID()).
				WithError(err).
				Error("Failed to kill the accserver process.")
		}
	}

	s.Live.ServerOffline()

	logrus.WithField("server_id", s.GetID()).Info("acc server stopped")

	return nil
}

func (s *Instance) GetProcessID() int {
	if s.IsRunning() {
		return s.cmd.Process.Pid
	}

	return 0
}

func (s *Instance) CanSaveSettings(aw AccWebSettingsJson, ac AccConfigFiles) error {
	if s.IsRunning() {
		return ErrServerCantBeRunning
	}

	if s.Cfg.Settings.EnableAdvWinCfg {
		if s.Cfg.Settings.AdvWindowsCfg == nil {
			return errors.New("where are the Advanced Windows Config definitions?")
		}

		if s.Cfg.Settings.AdvWindowsCfg.CoreAffinity > DefaultCoreAffinity {
			return ErrInvalidCoreAffinity
		}

		if _, ok := CpuPriorities[int(s.Cfg.Settings.AdvWindowsCfg.CpuPriority)]; !ok {
			return ErrInvalidCpuPriority
		}
	}

	return nil
}

func (s *Instance) Save() error {
	if err := s.CanSaveSettings(s.Cfg.Settings, s.AccCfg); err != nil {
		return err
	}

	if s.Cfg.Settings.AdvWindowsCfg != nil && s.Cfg.Settings.AdvWindowsCfg.CoreAffinity == 0 {
		s.Cfg.Settings.AdvWindowsCfg.CoreAffinity = DefaultCoreAffinity
	}

	fileList := map[string]interface{}{
		accwebConfigJsonName:  &s.Cfg,
		configurationJsonName: &s.AccCfg.Configuration,
		settingsJsonName:      &s.AccCfg.Settings,
		eventJsonName:         &s.AccCfg.Event,
		eventRulesJsonName:    &s.AccCfg.EventRules,
		entrylistJsonName:     &s.AccCfg.Entrylist,
		bopJsonName:           &s.AccCfg.Bop,
		assistRulesJsonName:   &s.AccCfg.AssistRules,
	}

	for filename, cfg := range fileList {
		if err := helper.SaveToPath(s.Path, filename, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (s *Instance) SaveAccWebConfig() error {
	s.Cfg.SetUpdateAt()
	return helper.SaveToPath(s.Path, accwebConfigJsonName, &s.Cfg)
}

func (s *Instance) CheckDirectory() error {
	fileList := []string{
		accwebConfigJsonName,
		configurationJsonName,
		settingsJsonName,
		eventJsonName,
		eventRulesJsonName,
		entrylistJsonName,
		bopJsonName,
		assistRulesJsonName,
		accDedicatedServerFile,
	}

	for _, filename := range fileList {
		p := path.Join(s.Path, filename)
		if !helper.Exists(p) {
			return fmt.Errorf("%w - missing '%s'", ErrServerDirIsInvalid, p)
		}
	}

	return nil
}

func (s *Instance) CheckServerExeMd5Sum() (bool, error) {
	sum, err := helper.CheckMd5Sum(path.Join(s.Path, accDedicatedServerFile))
	if err != nil {
		return false, err
	}

	r := false

	if s.Cfg.Md5Sum != sum {
		s.Cfg.Md5Sum = sum
		s.Cfg.SetUpdateAt()
		r = true
	}

	return r, nil
}

func (s *Instance) UpdateAccServerExe(srcFile string) (bool, error) {
	if s.IsRunning() {
		return false, ErrServerCantBeRunning
	}

	localFile := path.Join(s.Path, accDedicatedServerFile)

	if helper.Exists(localFile) {
		_ = os.Remove(localFile)
	}

	if err := helper.Copy(srcFile, localFile); err != nil {
		return false, err
	}

	if err := os.Chmod(localFile, 0755); err != nil {
		return false, err
	}

	// Copy all DLLs from the source server directory to the instance directory so Wine can load them.
	srcDir := filepath.Dir(srcFile)
	if files, err := os.ReadDir(srcDir); err == nil {
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".dll" {
				srcDll := filepath.Join(srcDir, f.Name())
				dstDll := filepath.Join(s.Path, f.Name())
				if helper.Exists(dstDll) {
					_ = os.Remove(dstDll)
				}
				_ = helper.Copy(srcDll, dstDll)
				_ = os.Chmod(dstDll, 0755)
			}
		}
	}

	return s.CheckServerExeMd5Sum()
}

func (s *Instance) IsRunning() bool {
	return s.cmd != nil && s.cmd.Process != nil && s.cmd.Process.Pid > 0 && s.cmd.ProcessState == nil
}

func (s *Instance) GetAccServerLogs() ([]byte, error) {
	logFilePath := path.Join(s.Path, accServerLogDir, accServerLogFile)
	if !helper.Exists(logFilePath) {
		return nil, errors.New("server log file doesn't exists")
	}

	return os.ReadFile(logFilePath)
}

func (s *Instance) ExportConfigFilesToZip() ([]byte, error) {
	fileList := map[string]interface{}{
		configurationJsonName: &s.AccCfg.Configuration,
		settingsJsonName:      &s.AccCfg.Settings,
		eventJsonName:         &s.AccCfg.Event,
		eventRulesJsonName:    &s.AccCfg.EventRules,
		entrylistJsonName:     &s.AccCfg.Entrylist,
		bopJsonName:           &s.AccCfg.Bop,
		assistRulesJsonName:   &s.AccCfg.AssistRules,
	}

	buf := new(bytes.Buffer)
	archive := zip.NewWriter(buf)

	for filename, obj := range fileList {
		l := logrus.WithField("filename", filename)

		contentData, err := helper.Encode(obj)
		if err != nil {
			l.WithError(err).Error("Error encoding config information")
			return nil, err
		}

		file, err := archive.Create(filename)
		if err != nil {
			l.WithError(err).Error("Error creating zip file")
			return nil, err
		}

		if _, err := file.Write(contentData); err != nil {
			l.WithError(err).Error("Error writing zip file")
			return nil, err
		}
	}

	if err := archive.Close(); err != nil {
		logrus.WithError(err).Error("Error closing zip file")
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *Instance) prepareInstanceDir() error {
	if err := s.CheckDirectory(); err != nil {
		return err
	}

	// Copy config files to cfg dir
	if err := helper.CreateIfNotExists(path.Join(s.Path, accCfgDir), 0755); err != nil {
		return err
	}

	fileList := []string{
		configurationJsonName,
		settingsJsonName,
		eventJsonName,
		eventRulesJsonName,
		entrylistJsonName,
		bopJsonName,
		assistRulesJsonName,
	}

	for _, name := range fileList {
		if err := helper.Copy(path.Join(s.Path, name), path.Join(s.Path, accCfgDir, name)); err != nil {
			return err
		}
	}

	// Ensure broadcasting.json is present in cfg for ACC dedicated server broadcasting
	bPort := 9000
	if s.AccCfg.Configuration.UdpPort > 1000 {
		bPort = s.AccCfg.Configuration.UdpPort - 600
	}
	bJson := map[string]interface{}{
		"updListenerPort":    bPort,
		"connectionPassword": "asd",
		"commandPassword":    "",
		"dumpLeaderboards":   0,
	}
	if bData, err := json.MarshalIndent(bJson, "", "  "); err == nil {
		_ = os.WriteFile(path.Join(s.Path, accCfgDir, "broadcasting.json"), bData, 0644)
	}

	return nil
}

func (s *Instance) wait() {
	// wait for shutdown or crash
	if err := s.cmd.Wait(); err != nil && err.Error() != "signal: killed" {
		logrus.WithError(err).Error("Error when server stopped")
	}

	_ = s.cmdOut.Close()

	event.EmmitEventInstanceStopped(s.ToEIB())
}

func (s *Instance) prepareCommandAndArgs() {
	command := "." + string(filepath.Separator) + accDedicatedServerFile
	var args []string

	if runtime.GOOS == "linux" && !cfg.SkipWine() {
		command = "wine"
		args = []string{accDedicatedServerFile}
	}

	cmd := exec.Command(command, args...)
	cmd.Dir = s.Path
	s.cmd = cmd
}

func (s *Instance) prepareCmdLogHandler() error {
	if s.cmd == nil {
		return errors.New("instance command not prepared")
	}

	var err error
	s.cmdOut, err = s.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	go func() {
		// Raw reader from process stdout
		raw := bufio.NewReader(s.cmdOut)

		// Detect UTF-16 LE BOM (0xFF 0xFE)
		isUtf16 := false
		if bom, err := raw.Peek(2); err == nil && len(bom) == 2 && bom[0] == 0xFF && bom[1] == 0xFE {
			// Discard BOM
			_, _ = raw.Discard(2)
			isUtf16 = true
		}

		var srcReader io.Reader = raw
		if isUtf16 {
			decoder := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
			srcReader = transform.NewReader(raw, decoder)
			logrus.Debug("Detected UTF-16LE output. Decoding to UTF-8.")
		}

		reader := bufio.NewReader(srcReader)
		var buffer bytes.Buffer
		readBuffer := make([]byte, 1024)

		for {
			n, err := reader.Read(readBuffer)
			if err != nil {
				if err == io.EOF {
					if buffer.Len() > 0 {
						s.processBufferedData(&buffer)
					}
					break
				}
				logrus.Warnf("Error while reading server console: %v", err)
				break
			}

			if n > 0 {
				buffer.Write(readBuffer[:n])
				s.processLinesFromBuffer(&buffer)
			}
		}
	}()

	return nil
}

func (s *Instance) processLinesFromBuffer(buffer *bytes.Buffer) {
	for {
		// Look for newline in buffer
		data := buffer.Bytes()
		newlineIndex := bytes.IndexByte(data, '\n')

		if newlineIndex == -1 {
			// No complete line found, keep data in buffer
			break
		}

		// Extract complete line (including newline)
		line := make([]byte, newlineIndex+1)
		buffer.Read(line)

		// Clean CRLF
		line = bytes.TrimRight(line, "\r\n")
		if len(line) == 0 {
			continue
		}

		decoded := helper.NormalizeEncoding(line)
		if len(decoded) > 0 && !shouldFilterOutput(decoded) {
			event.EmmitEventInstanceOutput(s.ToEIB(), decoded)
			if s.Collector != nil && s.Collector.IsActive() {
				s.Collector.HandleLogLine(string(decoded))
			}
		}
	}
}

func (s *Instance) processBufferedData(buffer *bytes.Buffer) {
	if buffer.Len() == 0 {
		return
	}

	// Process any remaining data as a single line
	data := buffer.Bytes()
	data = bytes.TrimRight(data, "\r\n")
	if len(data) > 0 {
		decoded := helper.NormalizeEncoding(data)
		if len(decoded) > 0 && !shouldFilterOutput(decoded) {
			event.EmmitEventInstanceOutput(s.ToEIB(), decoded)
			if s.Collector != nil && s.Collector.IsActive() {
				s.Collector.HandleLogLine(string(decoded))
			}
		}
	}

	buffer.Reset()
}

func (s *Instance) ToEIB() event.EventInstanceBase {
	t := s.AccCfg.Event.Track
	if s.Live.Track != "" {
		t = s.Live.Track
	}

	return event.NewEventInstanceBase(
		s.GetID(),
		s.AccCfg.Settings.ServerName,
		t,
		s.AccCfg.Configuration.TcpPort,
		s.AccCfg.Configuration.UdpPort,
		s.Live.NrClients,
	)
}

func shouldFilterOutput(data []byte) bool {
	for _, er := range outputFiltersEr {
		if er.Match(data) {
			return true
		}
	}

	return false
}
