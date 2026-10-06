package handlers

import (
	"time"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	db *gorm.DB
	tz *time.Location
}

func NewDashboardHandler(db *gorm.DB, tz *time.Location) *DashboardHandler {
	return &DashboardHandler{db: db, tz: tz}
}

// Stats Statistik dashboard admin
// @Summary Statistik dashboard admin
// @Tags Admin: Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} api.Envelope
// @Security BearerAuth
// @Router /admin/dashboard [get]
func (h *DashboardHandler) Stats(c *gin.Context) {
	now := time.Now().In(h.tz)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, h.tz)
	dayEnd := dayStart.Add(24 * time.Hour)

	var totalUsers, totalCourses, totalLocations, totalClasses, todayCount, hadirToday, terlambatToday, ditolakToday int64
	h.db.Model(&models.User{}).Count(&totalUsers)
	h.db.Model(&models.Course{}).Count(&totalCourses)
	h.db.Model(&models.Location{}).Count(&totalLocations)
	h.db.Model(&models.Class{}).Count(&totalClasses)
	h.db.Model(&models.Attendance{}).Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).Count(&todayCount)
	h.db.Model(&models.Attendance{}).Where("created_at >= ? AND created_at < ? AND status = ?", dayStart, dayEnd, models.StatusHadir).Count(&hadirToday)
	h.db.Model(&models.Attendance{}).Where("created_at >= ? AND created_at < ? AND status = ?", dayStart, dayEnd, models.StatusTerlambat).Count(&terlambatToday)
	h.db.Model(&models.Attendance{}).Where("created_at >= ? AND created_at < ? AND status = ?", dayStart, dayEnd, models.StatusDitolak).Count(&ditolakToday)

	var recentRows []models.Attendance
	h.db.Preload("User").
		Preload("Schedule", func(db *gorm.DB) *gorm.DB { return db.Unscoped().Preload("Course") }).
		Preload("Schedule.Course").
		Order("created_at DESC").Limit(5).Find(&recentRows)

	recent := make([]gin.H, 0, len(recentRows))
	for _, row := range recentRows {
		courseName := "-"
		if row.Schedule.ID > 0 {
			courseName = row.Schedule.Course.CourseName
		}
		recent = append(recent, gin.H{
			"id":         row.ID,
			"student":    row.User.Name,
			"course":     courseName,
			"status":     row.Status,
			"method":     row.Method,
			"created_at": row.CreatedAt.In(h.tz).Format("2006-01-02 15:04"),
		})
	}

	var scheduleRows []models.Schedule
	h.db.Preload("ClassRoom").
		Preload("Course").
		Preload("Location").
		Where("start_time >= ? AND start_time < ?", dayStart, dayEnd).
		Order("start_time ASC").Find(&scheduleRows)

	schedules := make([]gin.H, 0, len(scheduleRows))
	for _, s := range scheduleRows {
		schedules = append(schedules, gin.H{
			"id":        s.ID,
			"course":    s.Course.CourseName,
			"class":     s.ClassRoom.Name,
			"location":  s.Location.Name,
			"start":     s.StartTime.Format("15:04"),
			"end":       s.EndTime.Format("15:04"),
			"is_active": s.IsActive(now),
		})
	}

	api.OK(c, gin.H{
		"totals": gin.H{
			"users":     totalUsers,
			"courses":   totalCourses,
			"locations": totalLocations,
			"classes":   totalClasses,
			"today":     todayCount,
			"hadir":     hadirToday,
			"terlambat": terlambatToday,
			"ditolak":   ditolakToday,
		},
		"recent":    recent,
		"schedules": schedules,
	}, "Statistik dashboard")
}
