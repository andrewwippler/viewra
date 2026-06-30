package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/api/handlers"
)

func RegisterLiveTvRoutes(rg *gin.RouterGroup, handler *handlers.LiveTvHandler) {
	livetv := rg.Group("/livetv")
	livetv.GET("/:libraryId/channels", handler.ListChannels)
	livetv.GET("/:libraryId/epg", handler.ListEPG)
	livetv.GET("/:libraryId/epg/programs/:programId", handler.GetEPGProgram)
	livetv.POST("/:libraryId/scan-channels", handler.ScanChannels)
	livetv.POST("/:libraryId/scan-epg", handler.ScanEPG)
	livetv.GET("/:libraryId/mappings", handler.ListMappings)
	livetv.PUT("/:libraryId/mappings", handler.SetMapping)
	livetv.DELETE("/:libraryId/mappings/:xmltvChannelId", handler.DeleteMapping)
}
