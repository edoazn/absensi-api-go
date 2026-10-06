package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/dto"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CourseHandler struct {
	db *gorm.DB
}

func NewCourseHandler(db *gorm.DB) *CourseHandler {
	return &CourseHandler{db: db}
}

// Index Daftar mata kuliah
// @Summary Daftar mata kuliah
// @Tags Admin: Courses
// @Accept json
// @Produce json
// @Success 200 {object} api.Envelope
// @Security BearerAuth
// @Router /courses [get]
func (h *CourseHandler) Index(c *gin.Context) {
	var courses []models.Course
	if err := h.db.Order("course_code ASC").Find(&courses).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat mata kuliah")
		return
	}
	api.OK(c, courses, "Daftar mata kuliah")
}

func (h *CourseHandler) codeTaken(code string, excludeID uint) bool {
	var count int64
	q := h.db.Model(&models.Course{}).Where("course_code = ?", strings.ToUpper(strings.TrimSpace(code)))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

func applyCourseRequest(course *models.Course, req dto.CourseRequest) {
	course.CourseName = strings.TrimSpace(req.CourseName)
	course.CourseCode = strings.ToUpper(strings.TrimSpace(req.CourseCode))
	course.LecturerName = strings.TrimSpace(req.LecturerName)
	if req.LocationRoom == nil || strings.TrimSpace(*req.LocationRoom) == "" {
		course.LocationRoom = nil
	} else {
		room := strings.TrimSpace(*req.LocationRoom)
		course.LocationRoom = &room
	}
}

// Store Buat mata kuliah
// @Summary Buat mata kuliah
// @Tags Admin: Courses
// @Accept json
// @Produce json
// @Param body body dto.CourseRequest true "Mata kuliah"
// @Success 201 {object} api.Envelope
// @Failure 422 {object} api.Envelope
// @Security BearerAuth
// @Router /courses [post]
func (h *CourseHandler) Store(c *gin.Context) {
	var req dto.CourseRequest
	if !api.BindJSON(c, &req) {
		return
	}
	if h.codeTaken(req.CourseCode, 0) {
		api.Fail(c, http.StatusUnprocessableEntity, fmt.Sprintf("Kode %s sudah dipakai mata kuliah lain.", strings.ToUpper(strings.TrimSpace(req.CourseCode))))
		return
	}

	course := models.Course{}
	applyCourseRequest(&course, req)
	if err := h.db.Create(&course).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menyimpan mata kuliah")
		return
	}
	api.Created(c, course, "Mata kuliah berhasil ditambahkan")
}

// Update Ubah mata kuliah
// @Summary Ubah mata kuliah
// @Tags Admin: Courses
// @Accept json
// @Produce json
// @Param id path int true "ID mata kuliah"
// @Param body body dto.CourseRequest true "Mata kuliah"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /courses/{id} [put]
func (h *CourseHandler) Update(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		api.Fail(c, http.StatusBadRequest, "ID mata kuliah tidak valid")
		return
	}

	var course models.Course
	if err := h.db.First(&course, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Mata kuliah tidak ditemukan")
		return
	}

	var req dto.CourseRequest
	if !api.BindJSON(c, &req) {
		return
	}
	if h.codeTaken(req.CourseCode, id) {
		api.Fail(c, http.StatusUnprocessableEntity, fmt.Sprintf("Kode %s sudah dipakai mata kuliah lain.", strings.ToUpper(strings.TrimSpace(req.CourseCode))))
		return
	}

	applyCourseRequest(&course, req)
	if err := h.db.Save(&course).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menyimpan perubahan mata kuliah")
		return
	}
	api.OK(c, course, "Mata kuliah berhasil diperbarui")
}

// Destroy Hapus mata kuliah
// @Summary Hapus mata kuliah
// @Tags Admin: Courses
// @Accept json
// @Produce json
// @Param id path int true "ID mata kuliah"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /courses/{id} [delete]
func (h *CourseHandler) Destroy(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		api.Fail(c, http.StatusBadRequest, "ID mata kuliah tidak valid")
		return
	}
	if err := h.db.Delete(&models.Course{}, id).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menghapus mata kuliah")
		return
	}
	api.OK(c, nil, "Mata kuliah berhasil dihapus")
}
