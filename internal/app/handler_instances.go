package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/assetto-corsa-web/accweb/internal/pkg/instance"
	"github.com/assetto-corsa-web/accweb/internal/pkg/server_manager"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type ExtraAccSettings struct {
	PasswordIsEmpty          bool `json:"passwordIsEmpty"`
	AdminPasswordIsEmpty     bool `json:"adminPasswordIsEmpty"`
	SpectatorPasswordIsEmpty bool `json:"spectatorPasswordIsEmpty"`
}

type InstancePayload struct {
	ID               string                      `json:"id"`
	Path             string                      `json:"path"`
	IsRunning        bool                        `json:"is_running"`
	PID              int                         `json:"pid"`
	Settings         instance.AccWebSettingsJson `json:"accWeb"`
	AccSettings      instance.AccConfigFiles     `json:"acc"`
	AccExtraSettings ExtraAccSettings            `json:"accExtraSettings"`
}

type InstanceOS struct {
	Name   string `json:"name"`
	NumCPU int    `json:"numCpu"`
}

type SaveInstancePayload struct {
	AccWeb           instance.AccWebSettingsJson `json:"accWeb"`
	Acc              instance.AccConfigFiles     `json:"acc"`
	AccExtraSettings ExtraAccSettings            `json:"accExtraSettings"`
}

func NewInstancePayload(srv *instance.Instance) InstancePayload {
	res := InstancePayload{
		ID:          srv.GetID(),
		Path:        srv.Path,
		IsRunning:   srv.IsRunning(),
		PID:         srv.GetProcessID(),
		Settings:    srv.Cfg.Settings,
		AccSettings: srv.AccCfg,
	}

	res.AccExtraSettings.PasswordIsEmpty = res.AccSettings.Settings.Password == ""
	res.AccSettings.Settings.Password = ""

	res.AccExtraSettings.AdminPasswordIsEmpty = res.AccSettings.Settings.AdminPassword == ""
	res.AccSettings.Settings.AdminPassword = ""

	res.AccExtraSettings.SpectatorPasswordIsEmpty = res.AccSettings.Settings.SpectatorPassword == ""
	res.AccSettings.Settings.SpectatorPassword = ""

	return res
}

// GetInstance Get instance information
// @Summary Get acc instance information
// @Description Get acc instance information
// @Tags instances
// @Accept json
// @Produce json
// @Success 200 {object} InstancePayload
// @Failure 404
// @Param id path int true "Instance ID"
// @Router /instance/{id} [get]
// @Security JWT
func (h *Handler) GetInstance(c *gin.Context) {
	id := c.Param("id")

	srv, err := h.sm.GetServerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	res := NewInstancePayload(srv)

	c.JSON(http.StatusOK, res)
}

// NewInstance Create new instance information
// @Summary Create new acc instance information
// @Description Create new acc instance information
// @Tags instances
// @Accept json
// @Produce json
// @Success 200 {object} InstancePayload
// @Failure 400  {object} AccWError
// @Failure 500  {object} AccWError
// @Param instance body SaveInstancePayload true "Instance data"
// @Router /instance [post]
// @Security JWT
func (h *Handler) NewInstance(c *gin.Context) {
	var json SaveInstancePayload
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
		return
	}

	srv, err := h.sm.Create(&json.Acc, json.AccWeb)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	res := NewInstancePayload(srv)

	c.JSON(http.StatusCreated, res)
}

// SaveInstance Saves instance information
// @Summary Saves acc instance information
// @Description Saves acc instance information
// @Tags instances
// @Accept json
// @Produce json
// @Success 200 {object} InstancePayload
// @Failure 404
// @Failure 400 {object} AccWError
// @Failure 500 {object} AccWError
// @Param id path int true "Instance ID"
// @Param instance body SaveInstancePayload true "Instance data"
// @Router /instance/{id} [post]
// @Security JWT
func (h *Handler) SaveInstance(c *gin.Context) {
	id := c.Param("id")

	srv, err := h.sm.GetServerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	var json SaveInstancePayload
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
		return
	}

	if json.AccExtraSettings.PasswordIsEmpty {
		json.Acc.Settings.Password = ""
	} else if json.Acc.Settings.Password == "" {
		json.Acc.Settings.Password = srv.AccCfg.Settings.Password
	}

	if json.AccExtraSettings.SpectatorPasswordIsEmpty {
		json.Acc.Settings.SpectatorPassword = ""
	} else if json.Acc.Settings.SpectatorPassword == "" {
		json.Acc.Settings.SpectatorPassword = srv.AccCfg.Settings.SpectatorPassword
	}

	if json.AccExtraSettings.AdminPasswordIsEmpty {
		json.Acc.Settings.AdminPassword = ""
	} else if json.Acc.Settings.AdminPassword == "" {
		json.Acc.Settings.AdminPassword = srv.AccCfg.Settings.AdminPassword
	}

	if err := srv.CanSaveSettings(json.AccWeb, json.Acc); err != nil {
		c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
		return
	}

	// Preserve collector metadata if not explicitly provided (Rule #3)
	if json.AccWeb.EventID == "" && srv.Cfg.Settings.EventID != "" {
		json.AccWeb.EventID = srv.Cfg.Settings.EventID
	}
	if !json.AccWeb.CollectorEnabled && srv.Cfg.Settings.CollectorEnabled {
		json.AccWeb.CollectorEnabled = srv.Cfg.Settings.CollectorEnabled
	}
	if json.AccWeb.CollectorStatus == "" {
		json.AccWeb.CollectorStatus = srv.Cfg.Settings.CollectorStatus
	}

	srv.AccCfg = json.Acc
	srv.Cfg.Settings = json.AccWeb

	if err := srv.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	res := NewInstancePayload(srv)

	c.JSON(http.StatusCreated, res)
}

// DeleteInstance Delete instance
// @Summary Delete acc instance
// @Description Delete acc instance
// @Tags instances
// @Accept json
// @Produce json
// @Success 200
// @Failure 404
// @Failure 500 {object} AccWError
// @Param id path int true "Instance ID"
// @Router /instance/{id} [delete]
// @Security JWT
func (h *Handler) DeleteInstance(c *gin.Context) {
	id := c.Param("id")

	if err := h.sm.Delete(id); err != nil {
		if errors.Is(err, server_manager.ErrServerNotFound) {
			c.JSON(http.StatusNotFound, nil)
			return
		}

		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

// StartInstance Starts acc instance
// @Summary Starts acc instance
// @Description Starts acc instance
// @Tags instances
// @Accept json
// @Produce json
// @Success 200
// @Failure 404
// @Failure 400 {object} AccWError
// @Failure 500 {object} AccWError
// @Param id path int true "Instance ID"
// @Router /instance/{id}/start [post]
// @Security JWT
func (h *Handler) StartInstance(c *gin.Context) {
	id := c.Param("id")
	if err := h.sm.Start(id); err != nil {
		if errors.Is(err, server_manager.ErrServerNotFound) {
			c.JSON(http.StatusNotFound, nil)
			return
		}
		if errors.Is(err, instance.ErrServerCantBeRunning) {
			c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	if srv, err := h.sm.GetServerByID(id); err == nil {
		if srv.Cfg.Settings.EventID != "" && srv.Cfg.Settings.CollectorEnabled {
			go h.syncCollectorToBackend(srv.GetID(), srv.Cfg.Settings.EventID, true, "running")
		}
	}

	c.JSON(http.StatusOK, nil)
}

// StopInstance Stops acc instance
// @Summary Stops acc instance
// @Description Stops acc instance
// @Tags instances
// @Accept json
// @Produce json
// @Success 200
// @Failure 404
// @Failure 400 {object} AccWError
// @Failure 500 {object} AccWError
// @Param id path int true "Instance ID"
// @Router /instance/{id}/stop [post]
// @Security JWT
func (h *Handler) StopInstance(c *gin.Context) {
	id := c.Param("id")

	srv, err := h.sm.GetServerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	if err := srv.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	if srv.Cfg.Settings.EventID != "" {
		go h.syncCollectorToBackend(srv.GetID(), srv.Cfg.Settings.EventID, false, "stopped")
	}

	c.JSON(http.StatusOK, nil)
}

// CloneInstance Clones acc instance
// @Summary Clones acc instance
// @Description Clones acc instance
// @Tags instances
// @Accept json
// @Produce json
// @Success 200
// @Failure 404
// @Failure 500 {object} AccWError
// @Param id path int true "Instance ID"
// @Router /instance/{id}/clone [post]
// @Security JWT
func (h *Handler) CloneInstance(c *gin.Context) {
	id := c.Param("id")

	srv, err := h.sm.Duplicate(id)
	if err != nil {
		if errors.Is(err, server_manager.ErrServerNotFound) {
			c.JSON(http.StatusNotFound, nil)
			return
		}

		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	res := NewInstancePayload(srv)

	c.JSON(http.StatusOK, res)
}

type accWebInstanceLogs struct {
	ID   string `json:"id"`
	Logs string `json:"logs"`
}

// GetInstanceLogs Get acc instance logs
// @Summary Get acc instance logs
// @Description Get acc instance logs
// @Tags instances
// @Accept json
// @Produce json
// @Success 200 {object} accWebInstanceLogs
// @Failure 404
// @Failure 500 {object} AccWError
// @Param id path int true "Instance ID"
// @Router /instance/{id}/logs [get]
// @Security JWT
func (h *Handler) GetInstanceLogs(c *gin.Context) {
	id := c.Param("id")

	srv, err := h.sm.GetServerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	data, err := srv.GetAccServerLogs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, accWebInstanceLogs{ID: srv.GetID(), Logs: string(data)})
}

// ExportInstance Get acc instance configuration files
// @Summary Get acc instance configuration files
// @Description Get acc instance configuration files
// @Tags instances
// @Accept json
// @Produce json
// @Success 200 string filedata "Zip file with all cofiguration files"
// @Failure 404
// @Failure 500 {object} AccWError
// @Param id path int true "Instance ID"
// @Router /instance/{id}/export [get]
// @Security JWT
func (h *Handler) ExportInstance(c *gin.Context) {
	id := c.Param("id")

	srv, err := h.sm.GetServerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	data, err := srv.ExportConfigFilesToZip()
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"accweb_%s_cfg.zip\"", id))
	c.Data(http.StatusOK, "application/zip", data)
}

type LiveServerInstancePayload struct {
	ListServerItem
	Live *instance.LiveState `json:"live"`
}

// GetInstanceLiveState Get acc instance live information
// @Summary Get acc instance live information
// @Description Get acc instance live information
// @Tags instances
// @Accept json
// @Produce json
// @Success 200 {object} LiveServerInstancePayload
// @Failure 404
// @Param id path int true "Instance ID"
// @Router /instance/{id}/live [get]
// @Security JWT
func (h *Handler) GetInstanceLiveState(c *gin.Context) {
	id := c.Param("id")

	srv, err := h.sm.GetServerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	c.JSON(http.StatusOK, LiveServerInstancePayload{
		ListServerItem: buildListServerItem(srv),
		Live:           srv.Live,
	})
}

type UpdateCollectorPayload struct {
	EventID          string `json:"event_id"`
	CollectorEnabled bool   `json:"collector_enabled"`
}

type CollectorResponse struct {
	ServerID         string `json:"server_id"`
	EventID          string `json:"event_id"`
	CollectorEnabled bool   `json:"collector_enabled"`
	CollectorStatus  string `json:"collector_status"`
}

// GetCollectorSettings Get collector settings for an instance
// @Summary Get collector settings for an instance
// @Description Get collector settings for an instance
// @Tags instances
// @Accept json
// @Produce json
// @Success 200 {object} CollectorResponse
// @Failure 404
// @Param id path string true "Instance ID"
// @Router /instance/{id}/collector [get]
// @Security JWT
func (h *Handler) GetCollectorSettings(c *gin.Context) {
	id := c.Param("id")

	srv, err := h.sm.GetServerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	status := srv.Cfg.Settings.CollectorStatus
	if status == "" {
		status = srv.Cfg.Settings.DeriveCollectorStatus()
	}

	c.JSON(http.StatusOK, CollectorResponse{
		ServerID:         srv.GetID(),
		EventID:          srv.Cfg.Settings.EventID,
		CollectorEnabled: srv.Cfg.Settings.CollectorEnabled,
		CollectorStatus:  status,
	})
}

// SaveCollectorSettings Saves collector settings for an instance
// @Summary Saves collector settings for an instance
// @Description Saves collector settings for an instance
// @Tags instances
// @Accept json
// @Produce json
// @Success 200 {object} CollectorResponse
// @Failure 400 {object} AccWError
// @Failure 404
// @Failure 500 {object} AccWError
// @Param id path string true "Instance ID"
// @Param collector body UpdateCollectorPayload true "Collector settings"
// @Router /instance/{id}/collector [post]
// @Security JWT
func (h *Handler) SaveCollectorSettings(c *gin.Context) {
	id := c.Param("id")

	srv, err := h.sm.GetServerByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	var json UpdateCollectorPayload
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
		return
	}

	// Clean event_id: trim whitespace
	eventID := strings.TrimSpace(json.EventID)

	// Derive collector_status strictly based on collector_enabled (Rule #1)
	var collectorStatus string
	if json.CollectorEnabled {
		collectorStatus = instance.CollectorStatusEnabled
	} else {
		collectorStatus = instance.CollectorStatusStopped
	}

	srv.Cfg.Settings.EventID = eventID
	srv.Cfg.Settings.CollectorEnabled = json.CollectorEnabled
	srv.Cfg.Settings.CollectorStatus = collectorStatus

	// Save ONLY accwebConfig.json, leaving all 7 ACC config files untouched (Rule #2)
	if err := srv.SaveAccWebConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, CollectorResponse{
		ServerID:         srv.GetID(),
		EventID:          srv.Cfg.Settings.EventID,
		CollectorEnabled: srv.Cfg.Settings.CollectorEnabled,
		CollectorStatus:  srv.Cfg.Settings.CollectorStatus,
	})
}

func (h *Handler) syncCollectorToBackend(serverId, eventId string, enabled bool, status string) {
	if eventId == "" {
		return
	}

	apiUrl := "http://localhost:5000"
	if h.cfg != nil && h.cfg.BackendApiUrl != "" {
		apiUrl = h.cfg.BackendApiUrl
	}

	endpoint := fmt.Sprintf("%s/events/%s/collector", strings.TrimRight(apiUrl, "/"), eventId)

	payload := map[string]interface{}{
		"serverId":         serverId,
		"collectorEnabled": enabled,
		"collectorStatus":  status,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		logrus.WithError(err).Warn("Failed to marshal collector payload for backend sync")
		return
	}

	req, err := http.NewRequest(http.MethodPatch, endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		logrus.WithError(err).Warn("Failed to create HTTP request for backend sync")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logrus.WithError(err).Warnf("Failed to sync collector status to backend at %s", endpoint)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		logrus.Infof("Successfully synced collector status '%s' for event %s to backend", status, eventId)
	} else {
		logrus.Warnf("Backend returned HTTP status %d when syncing collector for event %s", resp.StatusCode, eventId)
	}
}
