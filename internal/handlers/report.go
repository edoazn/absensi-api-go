package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ReportHandler struct {
	db *gorm.DB
	tz *time.Location
}

func NewReportHandler(db *gorm.DB, tz *time.Location) *ReportHandler {
	return &ReportHandler{db: db, tz: tz}
}

type reportFilters struct {
	start      *time.Time
	end        *time.Time
	scheduleID uint
}

func (h *ReportHandler) buildQuery(filters reportFilters) *gorm.DB {
	query := h.db.Model(&models.Attendance{}).
		Preload("User").
		Preload("Schedule", func(db *gorm.DB) *gorm.DB { return db.Unscoped().Preload("Course").Preload("Location") }).
		Preload("Schedule.Course", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Schedule.Location", func(db *gorm.DB) *gorm.DB { return db.Unscoped() })

	if filters.start != nil {
		query = query.Where("created_at >= ?", *filters.start)
	}
	if filters.end != nil {
		query = query.Where("created_at < ?", filters.end.Add(24*time.Hour))
	}
	if filters.scheduleID > 0 {
		query = query.Where("schedule_id = ?", filters.scheduleID)
	}
	return query.Order("created_at DESC")
}

func parseReportQuery(c *gin.Context, tz *time.Location) (reportFilters, bool) {
	var filters reportFilters

	if raw := c.Query("start_date"); raw != "" {
		parsed, err := time.ParseInLocation("2006-01-02", raw, tz)
		if err != nil {
			api.FailWithErrors(c, http.StatusUnprocessableEntity, "Parameter tidak valid", map[string]string{
				"start_date": "format harus YYYY-MM-DD",
			})
			return filters, false
		}
		filters.start = &parsed
	}
	if raw := c.Query("end_date"); raw != "" {
		parsed, err := time.ParseInLocation("2006-01-02", raw, tz)
		if err != nil {
			api.FailWithErrors(c, http.StatusUnprocessableEntity, "Parameter tidak valid", map[string]string{
				"end_date": "format harus YYYY-MM-DD",
			})
			return filters, false
		}
		filters.end = &parsed
	}
	if raw := c.Query("schedule_id"); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id < 1 {
			api.FailWithErrors(c, http.StatusUnprocessableEntity, "Parameter tidak valid", map[string]string{
				"schedule_id": "harus angka positif",
			})
			return filters, false
		}
		filters.scheduleID = uint(id)
	}
	return filters, true
}

// AttendanceReport Laporan absensi (paginated)
// @Summary Laporan absensi (paginated)
// @Tags Admin: Reports
// @Accept json
// @Produce json
// @Param start_date query string false "YYYY-MM-DD"
// @Param end_date query string false "YYYY-MM-DD"
// @Param schedule_id query int false "ID jadwal"
// @Param page query int false "Halaman"
// @Param per_page query int false "Per halaman (1-100)"
// @Success 200 {object} api.Envelope
// @Security BearerAuth
// @Router /reports/attendance [get]
func (h *ReportHandler) AttendanceReport(c *gin.Context) {
	filters, ok := parseReportQuery(c, h.tz)
	if !ok {
		return
	}

	page := queryIntOr(c, "page", 1)
	perPage := queryIntOr(c, "per_page", 15)
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 15
	}

	var total int64
	if err := h.buildQuery(filters).Count(&total).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat laporan")
		return
	}

	var rows []models.Attendance
	if err := h.buildQuery(filters).
		Limit(perPage).
		Offset((page - 1) * perPage).
		Find(&rows).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat laporan")
		return
	}

	data := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item := map[string]any{
			"id":         row.ID,
			"student":    row.User.Name,
			"identity":   row.User.IdentityNumber,
			"status":     row.Status,
			"method":     row.Method,
			"distance":   row.Distance,
			"created_at": row.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if row.Schedule.ID > 0 {
			item["course"] = row.Schedule.Course.CourseName
			item["course_code"] = row.Schedule.Course.CourseCode
			item["location"] = row.Schedule.Location.Name
			item["schedule_start"] = row.Schedule.StartTime.Format("2006-01-02 15:04")
			item["schedule_end"] = row.Schedule.EndTime.Format("2006-01-02 15:04")
		}
		data = append(data, item)
	}

	lastPage := int(total) / perPage
	if int(total)%perPage > 0 {
		lastPage++
	}
	if lastPage < 1 {
		lastPage = 1
	}

	api.OK(c, gin.H{
		"items": data,
		"meta": gin.H{
			"current_page": page,
			"per_page":     perPage,
			"total":        total,
			"last_page":    lastPage,
		},
	}, "Laporan absensi")
}

// ExportExcel Export laporan absensi ke Excel
// @Summary Export laporan absensi ke Excel
// @Tags Admin: Reports
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param start_date query string false "YYYY-MM-DD"
// @Param end_date query string false "YYYY-MM-DD"
// @Param schedule_id query int false "ID jadwal"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /reports/attendance/export [get]
func (h *ReportHandler) ExportExcel(c *gin.Context) {
	filters, ok := parseReportQuery(c, h.tz)
	if !ok {
		return
	}

	var rows []models.Attendance
	if err := h.buildQuery(filters).Find(&rows).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat laporan")
		return
	}

	file := excelize.NewFile()
	sheet := "Absensi"
	file.SetSheetName("Sheet1", sheet)

	headers := []string{
		"ID", "Nama Mahasiswa", "NIM/NIP", "Mata Kuliah", "Kode MK", "Lokasi",
		"Waktu Jadwal", "Status", "Metode", "Jarak (m)", "Latitude", "Longitude", "Waktu Absensi",
	}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		file.SetCellValue(sheet, cell, header)
	}

	for i, row := range rows {
		excelRow := i + 2
		setCell := func(col int, value any) {
			cell, _ := excelize.CoordinatesToCellName(col, excelRow)
			file.SetCellValue(sheet, cell, value)
		}
		setCell(1, row.ID)
		setCell(2, row.User.Name)
		setCell(3, row.User.IdentityNumber)
		courseName, courseCode, locationName, scheduleWindow := "-", "-", "-", "-"
		if row.Schedule.ID > 0 {
			courseName = row.Schedule.Course.CourseName
			courseCode = row.Schedule.Course.CourseCode
			locationName = row.Schedule.Location.Name
			scheduleWindow = fmt.Sprintf("%s - %s",
				row.Schedule.StartTime.Format("02/01/2006 15:04"),
				row.Schedule.EndTime.Format("15:04"))
		}
		setCell(4, courseName)
		setCell(5, courseCode)
		setCell(6, locationName)
		setCell(7, scheduleWindow)
		statusLabel := "Ditolak"
		switch row.Status {
		case models.StatusHadir:
			statusLabel = "Hadir"
		case models.StatusTerlambat:
			statusLabel = "Terlambat"
		}
		setCell(8, statusLabel)
		setCell(9, methodLabel(row.Method))
		if row.Distance != nil {
			setCell(10, *row.Distance)
		} else {
			setCell(10, "")
		}
		if row.Latitude != nil {
			setCell(11, *row.Latitude)
		} else {
			setCell(11, "")
		}
		if row.Longitude != nil {
			setCell(12, *row.Longitude)
		} else {
			setCell(12, "")
		}
		setCell(13, row.CreatedAt.Format("02/01/2006 15:04:05"))
	}

	buf, err := file.WriteToBuffer()
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal membuat file Excel")
		return
	}

	filename := fmt.Sprintf("laporan-absensi-%s.xlsx", time.Now().In(h.tz).Format("20060102_150405"))
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func methodLabel(method string) string {
	switch method {
	case models.MethodGeolocation:
		return "Geolocation"
	case models.MethodQrCode:
		return "QR Code"
	case models.MethodAttendanceCode:
		return "Kode Manual"
	default:
		return method
	}
}
