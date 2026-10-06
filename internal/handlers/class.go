package handlers

import (
	"net/http"
	"strings"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/dto"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ClassHandler struct {
	db *gorm.DB
}

func NewClassHandler(db *gorm.DB) *ClassHandler {
	return &ClassHandler{db: db}
}

// Index Daftar kelas
// @Summary Daftar kelas
// @Tags Admin: Classes
// @Accept json
// @Produce json
// @Success 200 {object} api.Envelope
// @Security BearerAuth
// @Router /classes [get]
func (h *ClassHandler) Index(c *gin.Context) {
	var classes []models.Class
	if err := h.db.Preload("Students").Order("name ASC").Find(&classes).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat kelas")
		return
	}
	api.OK(c, classes, "Daftar kelas")
}

func (h *ClassHandler) syncStudents(tx *gorm.DB, classID uint, userIDs []uint) error {
	var class models.Class
	if err := tx.First(&class, classID).Error; err != nil {
		return err
	}
	var students []models.User
	if len(userIDs) > 0 {
		if err := tx.Find(&students, userIDs).Error; err != nil {
			return err
		}
	}
	return tx.Model(&class).Association("Students").Replace(students)
}

// Store Buat kelas
// @Summary Buat kelas
// @Tags Admin: Classes
// @Accept json
// @Produce json
// @Param body body dto.ClassRequest true "Kelas"
// @Success 201 {object} api.Envelope
// @Failure 422 {object} api.Envelope
// @Security BearerAuth
// @Router /classes [post]
func (h *ClassHandler) Store(c *gin.Context) {
	var req dto.ClassRequest
	if !api.BindJSON(c, &req) {
		return
	}

	class := models.Class{Name: strings.TrimSpace(req.Name), AcademicYear: strings.TrimSpace(req.AcademicYear)}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&class).Error; err != nil {
			return err
		}
		return h.syncStudents(tx, class.ID, req.UserIDs)
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menyimpan kelas")
		return
	}
	h.db.Preload("Students").First(&class, class.ID)
	recordAudit(h.db, c, "class.create", "class", &class.ID, map[string]any{"name": class.Name})
	api.Created(c, class, "Kelas berhasil ditambahkan")
}

// Update Ubah kelas
// @Summary Ubah kelas
// @Tags Admin: Classes
// @Accept json
// @Produce json
// @Param id path int true "ID kelas"
// @Param body body dto.ClassRequest true "Kelas"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /classes/{id} [put]
func (h *ClassHandler) Update(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		api.Fail(c, http.StatusBadRequest, "ID kelas tidak valid")
		return
	}

	var req dto.ClassRequest
	if !api.BindJSON(c, &req) {
		return
	}

	var class models.Class
	if err := h.db.First(&class, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Kelas tidak ditemukan")
		return
	}
	class.Name = strings.TrimSpace(req.Name)
	class.AcademicYear = strings.TrimSpace(req.AcademicYear)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&class).Error; err != nil {
			return err
		}
		return h.syncStudents(tx, class.ID, req.UserIDs)
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menyimpan perubahan kelas")
		return
	}
	h.db.Preload("Students").First(&class, id)
	recordAudit(h.db, c, "class.update", "class", &id, map[string]any{"name": class.Name})
	api.OK(c, class, "Kelas berhasil diperbarui")
}

// Destroy Hapus kelas
// @Summary Hapus kelas
// @Tags Admin: Classes
// @Accept json
// @Produce json
// @Param id path int true "ID kelas"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /classes/{id} [delete]
func (h *ClassHandler) Destroy(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		api.Fail(c, http.StatusBadRequest, "ID kelas tidak valid")
		return
	}
	var class models.Class
	if err := h.db.First(&class, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Kelas tidak ditemukan")
		return
	}
	if err := h.db.Delete(&models.Class{}, id).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menghapus kelas")
		return
	}
	recordAudit(h.db, c, "class.delete", "class", &id, map[string]any{"name": class.Name})
	api.OK(c, nil, "Kelas berhasil dihapus")
}
