package app

import (
	"net/http"
	"strconv"

	"github.com/assetto-corsa-web/accweb/internal/pkg/instance"
	"github.com/assetto-corsa-web/accweb/internal/pkg/server_manager"
	"github.com/gin-gonic/gin"
)

type GlobalEntry struct {
	Entries []instance.AccwebGlobalEntrylistJson `json:"entries"`
}

// ListServers Handle the list all ACC dedicated servers
// @Summary List all ACC dedicated servers
// @Schemes
// @Description List all ACC dedicated servers
// @Tags servers
// @Accept json
// @Produce json
// @Success 200 {object} []ListServerItem{}
// @Router /configure/global-admin [get]
// @Security JWT
func (h *Handler) ListGlobalAdmins(c *gin.Context) {
	list, err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryContextAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, GlobalEntry{Entries: list})
}

func (h *Handler) SaveGlobalAdmins(c *gin.Context) {
	var entry instance.AccwebGlobalEntrylistJson
	if err := c.BindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
		return
	}

	list, err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryContextAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	for _, e := range list {
		if e.PlayerId == entry.PlayerId {
			c.JSON(http.StatusBadRequest, newAccWError("player already exist"))
			return
		}
	}

	list = append(list, instance.AccwebGlobalEntrylistJson{
		PlayerName: entry.PlayerName,
		PlayerId:   entry.PlayerId,
	})

	if err := h.sm.SaveGlobalEntry(server_manager.GlobalEntryContextAdmin, list); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) RemoveGlobalAdmin(c *gin.Context) {
	idS := c.Param("id")
	id, err := strconv.Atoi(idS)
	if err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	list, err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryContextAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	if id >= len(list) {
		c.JSON(http.StatusBadRequest, newAccWError("invalid id"))
		return
	}

	list = append(list[:id], list[id+1:]...)

	if err := h.sm.SaveGlobalEntry(server_manager.GlobalEntryContextAdmin, list); err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) ListGlobalBans(c *gin.Context) {
	list, err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryContextBan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, GlobalEntry{Entries: list})
}

func (h *Handler) SaveGlobalBans(c *gin.Context) {
	var entry instance.AccwebGlobalEntrylistJson
	if err := c.BindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
		return
	}

	list, err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryContextBan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	for _, e := range list {
		if e.PlayerId == entry.PlayerId {
			c.JSON(http.StatusBadRequest, newAccWError("player already exist"))
			return
		}
	}

	list = append(list, instance.AccwebGlobalEntrylistJson{
		PlayerName: entry.PlayerName,
		PlayerId:   entry.PlayerId,
	})

	if err := h.sm.SaveGlobalEntry(server_manager.GlobalEntryContextBan, list); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) RemoveGlobalBans(c *gin.Context) {
	idS := c.Param("id")
	id, err := strconv.Atoi(idS)
	if err != nil {
		c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
		return
	}

	list, err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryContextBan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	if id >= len(list) {
		c.JSON(http.StatusBadRequest, newAccWError("invalid id"))
		return
	}

	list = append(list[:id], list[id+1:]...)

	if err := h.sm.SaveGlobalEntry(server_manager.GlobalEntryContextBan, list); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}
