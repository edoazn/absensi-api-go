package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/dto"
	"github.com/edoazn/absensi-go/internal/middleware"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/edoazn/absensi-go/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AttendanceHandler struct {
	db      *gorm.DB
	service *services.AttendanceService
	tz      *time.Location
}

func NewAttendanceHandler(db *gorm.DB, service *services.AttendanceService, tz *time.Location) *AttendanceHandler {
	return &AttendanceHandler{db: db, service: service, tz: tz}
}

// Store Kirim absensi (geolocation / qr_code / attendance_code)
// @Summary Kirim absensi (geolocation / qr_code / attendance_code)
// @Tags Attendance
// @Accept json
// @Produce json
// @Param body body dto.AttendanceRequest true "Data absensi"
// @Success 201 {object} api.Envelope
// @Failure 422 {object} api.Envelope
// @Failure 429 {object} api.Envelope
// @Security BearerAuth
// @Router /attendance [post]
func (h *AttendanceHandler) Store(c *gin.Context) {
	var req dto.AttendanceRequest
	if !api.BindJSON(c, &req) {
		return
	}

	var user models.User
	if err := h.db.First(&user, middleware.UserIDFrom(c)).Error; err != nil {
		api.Fail(c, http.StatusUnauthorized, "Unauthenticated")
		return
	}

	result := h.service.Process(&user, services.AttendanceInput{
		ScheduleID:     req.ScheduleID,
		Method:         req.Method,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		Accuracy:       req.Accuracy,
		QrToken:        req.QrToken,
		AttendanceCode: req.AttendanceCode,
	})

	if !result.Success {
		api.Fail(c, http.StatusUnprocessableEntity, result.Message)
		return
	}

	data := dto.AttendanceStoreData{
		Status:     result.Status,
		Distance:   result.Distance,
		Method:     result.Method,
		Message:    result.Message,
		Attendance: toAttendanceResponse(result.Attendance),
	}

	status := http.StatusOK
	if result.Status == models.StatusHadir || result.Status == models.StatusTerlambat {
		status = http.StatusCreated
	}
	c.JSON(status, api.Envelope{Success: true, Data: data, Message: result.Message})
}

// History Riwayat absensi mahasiswa
// @Summary Riwayat absensi mahasiswa
// @Tags Attendance
// @Accept json
// @Produce json
// @Param page query int false "Halaman"
// @Param per_page query int false "Jumlah per halaman"
// @Success 200 {object} api.Envelope
// @Security BearerAuth
// @Router /attendance/history [get]
func (h *AttendanceHandler) History(c *gin.Context) {
	page := queryIntOr(c, "page", 1)
	perPage := queryIntOr(c, "per_page", 15)
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 15
	}

	rows, total, err := h.service.History(middleware.UserIDFrom(c), page, perPage)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat riwayat absensi")
		return
	}

	items := make([]dto.AttendanceItemResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAttendanceItem(row))
	}

	lastPage := int(total) / perPage
	if int(total)%perPage > 0 {
		lastPage++
	}
	if lastPage < 1 {
		lastPage = 1
	}

	api.OK(c, gin.H{
		"items": items,
		"meta": gin.H{
			"current_page": page,
			"per_page":     perPage,
			"total":        total,
			"last_page":    lastPage,
		},
	}, "Riwayat absensi")
}

// TodaySchedules Jadwal hari ini
// @Summary Jadwal hari ini
// @Tags Attendance
// @Accept json
// @Produce json
// @Success 200 {object} api.Envelope
// @Security BearerAuth
// @Router /schedules/today [get]
func (h *AttendanceHandler) TodaySchedules(c *gin.Context) {
	var user models.User
	if err := h.db.First(&user, middleware.UserIDFrom(c)).Error; err != nil {
		api.Fail(c, http.StatusUnauthorized, "Unauthenticated")
		return
	}

	schedules, err := h.service.TodaySchedules(&user)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat jadwal hari ini")
		return
	}

	now := time.Now().In(h.tz)
	items := make([]gin.H, 0, len(schedules))
	for _, s := range schedules {
		items = append(items, gin.H{
			"id":          s.ID,
			"class":       s.ClassRoom.Name,
			"course":      s.Course.CourseName,
			"course_code": s.Course.CourseCode,
			"location":    s.Location.Name,
			"start_time":  s.StartTime.Format("2006-01-02 15:04:05"),
			"end_time":    s.EndTime.Format("2006-01-02 15:04:05"),
			"is_active":   s.IsActive(now),
		})
	}
	api.OK(c, items, "Jadwal hari ini")
}

func toAttendanceResponse(a *models.Attendance) dto.AttendanceResponse {
	return dto.AttendanceResponse{
		ID: a.ID, UserID: a.UserID, ScheduleID: a.ScheduleID,
		Latitude: a.Latitude, Longitude: a.Longitude, Distance: a.Distance,
		GpsAccuracy: a.GpsAccuracy,
		Status:      a.Status, Method: a.Method,
		CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func toAttendanceItem(a models.Attendance) dto.AttendanceItemResponse {
	return dto.AttendanceItemResponse{
		ID: a.ID, UserID: a.UserID, ScheduleID: a.ScheduleID,
		Latitude: a.Latitude, Longitude: a.Longitude, Distance: a.Distance,
		GpsAccuracy: a.GpsAccuracy,
		Status:      a.Status, Method: a.Method,
		CreatedAt: a.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

func queryIntOr(c *gin.Context, key string, fallback int) int {
	raw := c.Query(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
