package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	appLivetv "github.com/mantonx/viewra/internal/application/livetv"
)

type LiveTvHandler struct {
	listChannels  *appLivetv.ListChannelsUseCase
	listEPG       *appLivetv.ListEPGUseCase
	scanChannels  *appLivetv.ScanChannelsUseCase
	scanEPG       *appLivetv.ScanEPGUseCase
	listMappings  *appLivetv.ListMappingsUseCase
	setMapping    *appLivetv.SetMappingUseCase
	deleteMapping *appLivetv.DeleteMappingUseCase
}

func NewLiveTvHandler(
	listChannels *appLivetv.ListChannelsUseCase,
	listEPG *appLivetv.ListEPGUseCase,
	scanChannels *appLivetv.ScanChannelsUseCase,
	scanEPG *appLivetv.ScanEPGUseCase,
	listMappings *appLivetv.ListMappingsUseCase,
	setMapping *appLivetv.SetMappingUseCase,
	deleteMapping *appLivetv.DeleteMappingUseCase,
) *LiveTvHandler {
	return &LiveTvHandler{
		listChannels:  listChannels,
		listEPG:       listEPG,
		scanChannels:  scanChannels,
		scanEPG:       scanEPG,
		listMappings:  listMappings,
		setMapping:    setMapping,
		deleteMapping: deleteMapping,
	}
}

func (h *LiveTvHandler) ListChannels(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid library ID")
		return
	}

	resp, err := h.listChannels.Execute(c.Request.Context(), libraryID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *LiveTvHandler) ListEPG(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid library ID")
		return
	}

	fromStr := c.DefaultQuery("from", time.Now().Format(time.RFC3339))
	toStr := c.DefaultQuery("to", time.Now().Add(3*time.Hour).Format(time.RFC3339))

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_FROM", "Invalid from time")
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_TO", "Invalid to time")
		return
	}

	resp, err := h.listEPG.Execute(c.Request.Context(), libraryID, from, to)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *LiveTvHandler) ScanChannels(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid library ID")
		return
	}

	resp, err := h.scanChannels.Execute(c.Request.Context(), libraryID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *LiveTvHandler) ScanEPG(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid library ID")
		return
	}

	resp, err := h.scanEPG.Execute(c.Request.Context(), libraryID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *LiveTvHandler) GetEPGProgram(c *gin.Context) {
	programID, err := parseID(c.Param("programId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid program ID")
		return
	}

	program, err := h.listEPG.GetByID(c.Request.Context(), programID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, program)
}

func (h *LiveTvHandler) ListMappings(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid library ID")
		return
	}

	resp, err := h.listMappings.Execute(c.Request.Context(), libraryID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *LiveTvHandler) SetMapping(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid library ID")
		return
	}

	var req appLivetv.SetMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	resp, err := h.setMapping.Execute(c.Request.Context(), libraryID, req)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *LiveTvHandler) DeleteMapping(c *gin.Context) {
	libraryID, err := parseID(c.Param("libraryId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ID", "Invalid library ID")
		return
	}

	xmltvChannelID := c.Param("xmltvChannelId")
	if xmltvChannelID == "" {
		respondError(c, http.StatusBadRequest, "INVALID_CHANNEL_ID", "Missing XMLTV channel ID")
		return
	}

	if err := h.deleteMapping.Execute(c.Request.Context(), libraryID, xmltvChannelID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
