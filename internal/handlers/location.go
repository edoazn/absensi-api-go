package handlers

import (
	"net/http"
	"strconv"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/dto"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LocationHandler struct {
	db *gorm.DB
}

func NewLocationHandler(db *gorm.DB) *LocationHandler {
	return &LocationHandler{db: db}
}

// Index Daftar lokasi
// @Summary Daftar lokasi
// @Tags Admin: Locations
// @Accept json
// @Produce json
// @Success 200 {object} api.Envelope
// @Security BearerAuth
// @Router /locations [get]
func (h *LocationHandler) Index(c *gin.Context) {
	var locations []models.Location
	if err := h.db.Order("name ASC").Find(&locations).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat lokasi")
		return
	}
	api.OK(c, locations, "Daftar lokasi")
}

// Store Buat lokasi
// @Summary Buat lokasi
// @Tags Admin: Locations
// @Accept json
// @Produce json
// @Param body body dto.LocationRequest true "Lokasi"
// @Success 201 {object} api.Envelope
// @Failure 422 {object} api.Envelope
// @Security BearerAuth
// @Router /locations [post]
func (h *LocationHandler) Store(c *gin.Context) {
	var req dto.LocationRequest
	if !api.BindJSON(c, &req) {
		return
	}

	location := models.Location{
		Name:      req.Name,
		Latitude:  *req.Latitude,
		Longitude: *req.Longitude,
		Radius:    *req.Radius,
	}
	if err := h.db.Create(&location).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menyimpan lokasi")
		return
	}
	api.Created(c, location, "Lokasi berhasil ditambahkan")
}

// Update Ubah lokasi
// @Summary Ubah lokasi
// @Tags Admin: Locations
// @Accept json
// @Produce json
// @Param id path int true "ID lokasi"
// @Param body body dto.LocationRequest true "Lokasi"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /locations/{id} [put]
func (h *LocationHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		api.Fail(c, http.StatusBadRequest, "ID lokasi tidak valid")
		return
	}

	var location models.Location
	if err := h.db.First(&location, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Lokasi tidak ditemukan")
		return
	}

	var req dto.LocationRequest
	if !api.BindJSON(c, &req) {
		return
	}

	location.Name = req.Name
	location.Latitude = *req.Latitude
	location.Longitude = *req.Longitude
	location.Radius = *req.Radius

	if err := h.db.Save(&location).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memperbarui lokasi")
		return
	}
	api.OK(c, location, "Lokasi berhasil diperbarui")
}
