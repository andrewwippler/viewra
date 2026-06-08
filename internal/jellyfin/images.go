package jellyfin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mantonx/viewra/internal/domain/images"
)

// GetItemImageInfo handles GET /Items/:itemId/Images
func (h *Handler) GetItemImageInfo(c *gin.Context) {
	itemID, err := parseInt64(c.Param("itemId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	imgs, err := h.imagesGetMedia.Execute(c.Request.Context(), int(itemID))
	if err != nil || imgs == nil {
		c.JSON(http.StatusOK, []ImageInfo{})
		return
	}

	infos := make([]ImageInfo, 0, len(imgs.Images))
	for i, img := range imgs.Images {
		jfType := mapViewraImageTypeToJellyfin(img.ImageType)
		if jfType == "" {
			continue
		}
		w, h := 0, 0
		if img.Width != nil {
			w = *img.Width
		}
		if img.Height != nil {
			h = *img.Height
		}
		info := ImageInfo{
			ImageType: jfType,
			Width:     w,
			Height:    h,
		}
		if i > 0 {
			idx := i
			info.ImageIndex = &idx
		}
		infos = append(infos, info)
	}

	c.JSON(http.StatusOK, infos)
}

// GetItemImage handles GET /Items/:itemId/Images/:imageType
func (h *Handler) GetItemImage(c *gin.Context) {
	itemID, err := parseInt64(c.Param("itemId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	imageType := c.Param("imageType")
	viewraType := mapJellyfinImageTypeToViewra(imageType)
	if viewraType == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unsupported image type: " + imageType})
		return
	}

	imgs, err := h.imagesGetMedia.Execute(c.Request.Context(), int(itemID))
	if err != nil || imgs == nil || len(imgs.Images) == 0 {
		// Try entity images for TV shows/collections
		entityImgs, err2 := h.imagesGetEntity.Execute(c.Request.Context(), images.MediaTypeTVShow, int(itemID))
		if err2 != nil || entityImgs == nil || len(entityImgs.Images) == 0 {
			// Try as movie entity
			entityImgs, err2 = h.imagesGetEntity.Execute(c.Request.Context(), images.MediaTypeMovie, int(itemID))
			if err2 != nil || entityImgs == nil || len(entityImgs.Images) == 0 {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Image not found"})
				return
			}
		}
		imgs = entityImgs
	}

	// Find matching image type
	for _, img := range imgs.Images {
		if img.ImageType == viewraType || (viewraType == string(images.ImageTypePoster) && img.ImageType == string(images.ImageTypePoster)) {
			filePath := img.FilePath
			if img.LocalCachePath != nil && *img.LocalCachePath != "" {
				filePath = *img.LocalCachePath
			}
			if filePath == "" {
				continue
			}
			c.Header("Cache-Control", "public, max-age=86400")
			c.File(filePath)
			return
		}
	}

	// Fallback: serve first available image
	for _, img := range imgs.Images {
		filePath := img.FilePath
		if img.LocalCachePath != nil && *img.LocalCachePath != "" {
			filePath = *img.LocalCachePath
		}
		if filePath != "" {
			c.Header("Cache-Control", "public, max-age=86400")
			c.File(filePath)
			return
		}
	}

	c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Image not found"})
}

// GetItemImageByIndex handles GET /Items/:itemId/Images/:imageType/:index
func (h *Handler) GetItemImageByIndex(c *gin.Context) {
	itemID, err := parseInt64(c.Param("itemId"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	imageType := c.Param("imageType")
	indexStr := c.Param("index")
	index, _ := strconv.Atoi(indexStr)
	_ = index

	viewraType := mapJellyfinImageTypeToViewra(imageType)
	if viewraType == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unsupported image type"})
		return
	}

	imgs, err := h.imagesGetMedia.Execute(c.Request.Context(), int(itemID))
	if err != nil || imgs == nil || len(imgs.Images) == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Find matching images of the requested type
	matchingImgs := make([]ImageInfo, 0)
	for _, img := range imgs.Images {
		jfType := mapViewraImageTypeToJellyfin(img.ImageType)
		if jfType == imageType {
		w, h := 0, 0
		if img.Width != nil {
			w = *img.Width
		}
		if img.Height != nil {
			h = *img.Height
		}
		matchingImgs = append(matchingImgs, ImageInfo{
			ImageType: jfType,
			Width:     w,
			Height:    h,
			Path:      img.FilePath,
		})
		}
	}

	if index >= len(matchingImgs) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	target := matchingImgs[index]
	if target.Path == "" {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.Header("Cache-Control", "public, max-age=86400")
	c.File(target.Path)
}
