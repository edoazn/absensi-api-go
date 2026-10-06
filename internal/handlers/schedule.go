package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/dto"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/edoazn/absensi-go/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type ScheduleHandler struct {
	db          *gorm.DB
	scheduleSvc *services.ScheduleService
	tz          *time.Location
}

func NewScheduleHandler(db *gorm.DB, scheduleSvc *services.ScheduleService, tz *time.Location) *ScheduleHandler {
	return &ScheduleHandler{db: db, scheduleSvc: scheduleSvc, tz: tz}
}

// Index Daftar jadwal
// @Summary Daftar jadwal
// @Tags Admin: Schedules
// @Accept json
// @Produce json
// @Param class_id query int false "Filter kelas"
// @Param course_id query int false "Filter mata kuliah"
// @Success 200 {object} api.Envelope
// @Security BearerAuth
// @Router /schedules [get]
func (h *ScheduleHandler) Index(c *gin.Context) {
	var schedules []models.Schedule
	query := h.db.
		Preload("ClassRoom", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Course", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Location", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("start_time DESC")

	if classID := c.Query("class_id"); classID != "" {
		query = query.Where("class_id = ?", classID)
	}
	if courseID := c.Query("course_id"); courseID != "" {
		query = query.Where("course_id = ?", courseID)
	}

	now := time.Now().In(h.tz)
	items := make([]gin.H, 0)
	if err := query.Find(&schedules).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat jadwal")
		return
	}
	for _, s := range schedules {
		items = append(items, scheduleItem(s, now))
	}
	api.OK(c, items, "Daftar jadwal")
}

// Show Detail jadwal
// @Summary Detail jadwal
// @Tags Admin: Schedules
// @Accept json
// @Produce json
// @Param id path int true "ID jadwal"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /schedules/{id} [get]
func (h *ScheduleHandler) Show(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		api.Fail(c, http.StatusBadRequest, "ID jadwal tidak valid")
		return
	}

	var schedule models.Schedule
	if err := h.db.
		Preload("ClassRoom", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Course", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Location", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		First(&schedule, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Jadwal tidak ditemukan")
		return
	}
	api.OK(c, scheduleItem(schedule, time.Now().In(h.tz)), "Detail jadwal")
}

func scheduleItem(s models.Schedule, now time.Time) gin.H {
	var recurrenceEnd any
	if s.RecurrenceEnd != nil {
		recurrenceEnd = s.RecurrenceEnd.Format("2006-01-02")
	}
	return gin.H{
		"id":              s.ID,
		"class_id":        s.ClassID,
		"course_id":       s.CourseID,
		"location_id":     s.LocationID,
		"class":           s.ClassRoom.Name,
		"course":          s.Course.CourseName,
		"course_code":     s.Course.CourseCode,
		"location":        s.Location.Name,
		"start_time":      s.StartTime.Format("2006-01-02 15:04:05"),
		"end_time":        s.EndTime.Format("2006-01-02 15:04:05"),
		"is_recurring":    s.IsRecurring,
		"repeat_days":     s.RepeatDaySlice(),
		"recurrence_end":  recurrenceEnd,
		"is_active":       s.IsActive(now),
		"attendance_code": s.AttendanceCode,
		"code_expires_at": formatNullableTime(s.CodeExpiresAt),
		"has_qr":          s.QrToken != nil,
	}
}

// Store Buat jadwal
// @Summary Buat jadwal
// @Tags Admin: Schedules
// @Accept json
// @Produce json
// @Param body body dto.ScheduleRequest true "Jadwal"
// @Success 201 {object} api.Envelope
// @Failure 422 {object} api.Envelope
// @Security BearerAuth
// @Router /schedules [post]
func (h *ScheduleHandler) Store(c *gin.Context) {
	var req dto.ScheduleRequest
	if !api.BindJSON(c, &req) {
		return
	}

	for _, check := range []struct {
		model any
		id    uint
		msg   string
	}{
		{&models.Class{}, req.ClassID, "Kelas tidak ditemukan"},
		{&models.Course{}, req.CourseID, "Mata kuliah tidak ditemukan"},
		{&models.Location{}, req.LocationID, "Lokasi tidak ditemukan"},
	} {
		if err := h.db.First(check.model, check.id).Error; err != nil {
			api.Fail(c, http.StatusUnprocessableEntity, check.msg)
			return
		}
	}

	start, ok1 := parseFlexibleTime(req.StartTime, h.tz)
	end, ok2 := parseFlexibleTime(req.EndTime, h.tz)
	if !ok1 || !ok2 {
		api.Fail(c, http.StatusUnprocessableEntity, "Format waktu tidak valid (gunakan YYYY-MM-DD HH:MM:SS)")
		return
	}
	if !end.After(start) {
		api.FailWithErrors(c, http.StatusUnprocessableEntity, "Data tidak valid", map[string]string{
			"end_time": "waktu selesai harus setelah waktu mulai",
		})
		return
	}

	schedule := models.Schedule{
		ClassID:    req.ClassID,
		CourseID:   req.CourseID,
		LocationID: req.LocationID,
		StartTime:  start,
		EndTime:    end,
	}
	if !applyRecurrence(c, &schedule, req, h.tz) {
		return
	}
	if err := h.db.Create(&schedule).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menyimpan jadwal")
		return
	}
	api.Created(c, gin.H{"id": schedule.ID}, "Jadwal berhasil ditambahkan")
}

// applyRecurrence memvalidasi lalu menerapkan field pengulangan mingguan.
// Mengembalikan false bila kombinasi tidak valid (respons 422 sudah dikirim).
func applyRecurrence(c *gin.Context, schedule *models.Schedule, req dto.ScheduleRequest, tz *time.Location) bool {
	if !req.IsRecurring {
		schedule.IsRecurring = false
		schedule.RepeatDays = nil
		schedule.RecurrenceEnd = nil
		return true
	}
	if len(req.RepeatDays) == 0 {
		api.FailWithErrors(c, http.StatusUnprocessableEntity, "Data tidak valid", map[string]string{
			"repeat_days": "wajib dipilih minimal satu hari saat jadwal berulang",
		})
		return false
	}

	set := make(map[int]bool, len(req.RepeatDays))
	days := make([]int, 0, len(req.RepeatDays))
	for _, d := range req.RepeatDays {
		if !set[d] {
			set[d] = true
			days = append(days, d)
		}
	}
	sort.Ints(days)
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = strconv.Itoa(d)
	}
	csv := strings.Join(parts, ",")
	schedule.IsRecurring = true
	schedule.RepeatDays = &csv

	if req.RecurrenceEnd != "" {
		if end, err := time.ParseInLocation("2006-01-02", req.RecurrenceEnd, tz); err == nil {
			schedule.RecurrenceEnd = &end
		}
	}
	return true
}

// Update Ubah jadwal
// @Summary Ubah jadwal
// @Tags Admin: Schedules
// @Accept json
// @Produce json
// @Param id path int true "ID jadwal"
// @Param body body dto.ScheduleRequest true "Jadwal"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Failure 422 {object} api.Envelope
// @Security BearerAuth
// @Router /schedules/{id} [put]
func (h *ScheduleHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		api.Fail(c, http.StatusBadRequest, "ID jadwal tidak valid")
		return
	}

	var schedule models.Schedule
	if err := h.db.First(&schedule, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Jadwal tidak ditemukan")
		return
	}

	var req dto.ScheduleRequest
	if !api.BindJSON(c, &req) {
		return
	}

	for _, check := range []struct {
		model any
		id    uint
		msg   string
	}{
		{&models.Class{}, req.ClassID, "Kelas tidak ditemukan"},
		{&models.Course{}, req.CourseID, "Mata kuliah tidak ditemukan"},
		{&models.Location{}, req.LocationID, "Lokasi tidak ditemukan"},
	} {
		if err := h.db.First(check.model, check.id).Error; err != nil {
			api.Fail(c, http.StatusUnprocessableEntity, check.msg)
			return
		}
	}

	start, ok1 := parseFlexibleTime(req.StartTime, h.tz)
	end, ok2 := parseFlexibleTime(req.EndTime, h.tz)
	if !ok1 || !ok2 {
		api.Fail(c, http.StatusUnprocessableEntity, "Format waktu tidak valid (gunakan YYYY-MM-DD HH:MM:SS)")
		return
	}
	if !end.After(start) {
		api.FailWithErrors(c, http.StatusUnprocessableEntity, "Data tidak valid", map[string]string{
			"end_time": "waktu selesai harus setelah waktu mulai",
		})
		return
	}

	schedule.ClassID = req.ClassID
	schedule.CourseID = req.CourseID
	schedule.LocationID = req.LocationID
	schedule.StartTime = start
	schedule.EndTime = end
	if !applyRecurrence(c, &schedule, req, h.tz) {
		return
	}

	if err := h.db.Save(&schedule).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memperbarui jadwal")
		return
	}
	api.OK(c, gin.H{"id": schedule.ID}, "Jadwal berhasil diperbarui")
}

// Destroy Hapus jadwal (soft delete)
// @Summary Hapus jadwal (soft delete)
// @Tags Admin: Schedules
// @Accept json
// @Produce json
// @Param id path int true "ID jadwal"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /schedules/{id} [delete]
func (h *ScheduleHandler) Destroy(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		api.Fail(c, http.StatusBadRequest, "ID jadwal tidak valid")
		return
	}

	var schedule models.Schedule
	if err := h.db.First(&schedule, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Jadwal tidak ditemukan")
		return
	}
	if err := h.db.Delete(&models.Schedule{}, id).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menghapus jadwal")
		return
	}
	api.OK(c, nil, "Jadwal berhasil dihapus")
}

// GenerateCode Generate kode absensi
// @Summary Generate kode absensi
// @Tags Admin: Schedules
// @Accept json
// @Produce json
// @Param id path int true "ID jadwal"
// @Param body body dto.GenerateCodeRequest false "Masa berlaku (menit, 1-1440)"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /schedules/{id}/generate-code [post]
func (h *ScheduleHandler) GenerateCode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		api.Fail(c, http.StatusBadRequest, "ID jadwal tidak valid")
		return
	}

	var req dto.GenerateCodeRequest
	raw, _ := c.GetRawData()
	if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 {
		if err := json.Unmarshal(trimmed, &req); err != nil {
			api.FailWithErrors(c, http.StatusUnprocessableEntity, "Format JSON tidak valid", map[string]string{
				"body": "harus berupa JSON object",
			})
			return
		}
		if req.MinutesValid != nil && (*req.MinutesValid < 1 || *req.MinutesValid > 1440) {
			api.FailWithErrors(c, http.StatusUnprocessableEntity, "Parameter tidak valid", map[string]string{
				"minutes_valid": "harus antara 1 dan 1440",
			})
			return
		}
	}

	minutesValid := 30
	if req.MinutesValid != nil {
		minutesValid = *req.MinutesValid
	}

	var schedule models.Schedule
	if err := h.db.First(&schedule, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Jadwal tidak ditemukan")
		return
	}

	code, expiresAt, err := h.scheduleSvc.GenerateAttendanceCode(uint(id), minutesValid)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal membuat kode absensi")
		return
	}
	api.OK(c, gin.H{
		"code":          code,
		"expires_at":    expiresAt.Format("2006-01-02 15:04:05"),
		"minutes_valid": minutesValid,
	}, "Kode absensi berhasil dibuat")
}

// GenerateQr Rotate token QR jadwal
// @Summary Rotate token QR jadwal
// @Tags Admin: Schedules
// @Accept json
// @Produce json
// @Param id path int true "ID jadwal"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /schedules/{id}/generate-qr [post]
func (h *ScheduleHandler) GenerateQr(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		api.Fail(c, http.StatusBadRequest, "ID jadwal tidak valid")
		return
	}

	var schedule models.Schedule
	if err := h.db.First(&schedule, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Jadwal tidak ditemukan")
		return
	}

	token, err := h.scheduleSvc.RegenerateQrToken(uint(id))
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal membuat QR token")
		return
	}
	api.OK(c, gin.H{"qr_token": token}, "QR token berhasil dibuat ulang, token lama tidak berlaku")
}

// QrPng Gambar QR PNG jadwal
// @Summary Gambar QR PNG jadwal
// @Tags Admin: Schedules
// @Produce image/png
// @Param id path int true "ID jadwal"
// @Success 200 {file} file
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /schedules/{id}/qr.png [get]
func (h *ScheduleHandler) QrPng(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		http.NotFound(c.Writer, c.Request)
		return
	}
	var sched models.Schedule
	if err := h.db.First(&sched, id).Error; err != nil || sched.QrToken == nil || *sched.QrToken == "" {
		http.NotFound(c.Writer, c.Request)
		return
	}
	png, err := qrcode.Encode(*sched.QrToken, qrcode.Medium, 320)
	if err != nil {
		http.Error(c.Writer, "gagal membuat QR", http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/png", png)
}

func parseFlexibleTime(raw string, tz *time.Location) (time.Time, bool) {
	layouts := []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02T15:04"}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, strings.TrimSpace(raw), tz); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func formatNullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02 15:04:05")
}
