package app

import (
	"net/http"
	"strconv"

	"github.com/assetto-corsa-web/accweb/internal/pkg/instance"
	"github.com/assetto-corsa-web/accweb/internal/pkg/server_manager"
	"github.com/gin-gonic/gin"
)

// ListGlobalAdmins List all global admins
// @Summary List all global admins
// @Schemes
// @Description List all global admins
// @Tags servers
// @Accept json
// @Produce json
// @Success 200 {object} instance.AccwebGlobalEntrylistJson{}
// @Failure 500 {object} AccWError
// @Router /configure/global-entrylist [get]
// @Security JWT
func (h *Handler) ListGlobalAdmins(c *gin.Context) {
	var data instance.AccwebGlobalEntrylistJson
	if err := h.sm.LoadGlobalEntry(server_manager.GlobalListCtxEntry, &data); err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, data)
}

// SaveGlobalAdmins Save global admins
// @Summary Save global admins
// @Schemes
// @Description Save global admins
// @Tags servers
// @Accept json
// @Produce json
// @Param payload body instance.AccwebGlobalEntrylistJson true "Global admins payload"
// @Success 200
// @Failure 400 {object} AccWError
// @Failure 500 {object} AccWError
// @Router /configure/global-entrylist [post]
// @Security JWT
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

// ListGlobalBans List all global bans
// @Summary List all global bans
// @Schemes
// @Description List all global bans
// @Tags servers
// @Accept json
// @Produce json
// @Success 200 {object} instance.AccwebGlobalBanlistJson{}
// @Failure 500 {object} AccWError
// @Router /configure/global-ban [get]
// @Security JWT
func (h *Handler) ListGlobalBans(c *gin.Context) {
	var data instance.AccwebGlobalBanlistJson
	err := h.sm.LoadGlobalEntry(server_manager.GlobalEntryCtxBan, &data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newAccWError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, data)
}

// SaveGlobalBans Save global bans
// @Summary Save global bans
// @Schemes
// @Description Save global bans
// @Tags servers
// @Accept json
// @Produce json
// @Param payload body instance.AccwebGlobalBanEntryJson true "Global ban entry payload"
// @Success 200
// @Failure 400 {object} AccWError
// @Failure 500 {object} AccWError
// @Router /configure/global-ban [post]
// @Security JWT
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

// EnableToggleGlobalBans Toggle global bans enabled status
// @Summary Toggle global bans enabled status
// @Schemes
// @Description Toggle global bans enabled status
// @Tags servers
// @Accept json
// @Produce json
// @Success 200
// @Failure 500 {object} AccWError
// @Router /configure/global-ban/enable-toggle [post]
// @Security JWT
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

// RemoveGlobalBans Remove a global ban entry
// @Summary Remove a global ban entry
// @Schemes
// @Description Remove a global ban entry by ID
// @Tags servers
// @Accept json
// @Produce json
// @Param id path int true "Ban entry ID"
// @Success 200
// @Failure 400 {object} AccWError
// @Failure 500 {object} AccWError
// @Router /configure/global-ban/{id} [delete]
// @Security JWT
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
