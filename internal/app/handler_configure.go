package app

import (
	"net/http"
	"strconv"

	"github.com/assetto-corsa-web/accweb/internal/pkg/instance"
	"github.com/assetto-corsa-web/accweb/internal/pkg/server_manager"
	"github.com/gin-gonic/gin"
)

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
	var data instance.AccwebGlobalEntrylistJson
	if err := h.sm.LoadGlobalEntry(server_manager.GlobalListCtxEntry, &data); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) SaveGlobalAdmins(c *gin.Context) {
	var payload instance.AccwebGlobalEntrylistJson
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
		return
	}

	if err := h.sm.SaveGlobalEntry(server_manager.GlobalListCtxEntry, payload); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) ListGlobalBans(c *gin.Context) {
	var data instance.AccwebGlobalBanlistJson
	err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryCtxBan, &data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) SaveGlobalBans(c *gin.Context) {
	var entry instance.AccwebGlobalBanEntryJson
	if err := c.BindJSON(&entry); err != nil {
		c.JSON(http.StatusBadRequest, newAccWError(err.Error()))
		return
	}

	var data instance.AccwebGlobalBanlistJson
	err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryCtxBan, &data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	for _, e := range data.Entries {
		if e.PlayerId == entry.PlayerId {
			c.JSON(http.StatusBadRequest, newAccWError("player already exist"))
			return
		}
	}

	data.Entries = append(data.Entries, entry)

	if err := h.sm.SaveGlobalEntry(server_manager.GlobalEntryCtxBan, data); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}

func (h *Handler) EnableToggleGlobalBans(c *gin.Context) {
	var data instance.AccwebGlobalBanlistJson
	err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryCtxBan, &data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	data.Enabled = !data.Enabled

	if err := h.sm.SaveGlobalEntry(server_manager.GlobalEntryCtxBan, data); err != nil {
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

	var data instance.AccwebGlobalBanlistJson
	if err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryCtxBan, &data); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	if id >= len(data.Entries) {
		c.JSON(http.StatusBadRequest, newAccWError("invalid id"))
		return
	}

	data.Entries = append(data.Entries[:id], data.Entries[id+1:]...)

	if err := h.sm.SaveGlobalEntry(server_manager.GlobalEntryCtxBan, data); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, nil)
}
