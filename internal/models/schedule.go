package models

import (
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Schedule struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	ClassID        uint           `json:"class_id"`
	CourseID       uint           `json:"course_id"`
	LocationID     uint           `json:"location_id"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        time.Time      `json:"end_time"`
	IsRecurring    bool           `gorm:"column:is_recurring" json:"is_recurring"`
	RepeatDays     *string        `gorm:"column:repeat_days" json:"-"`
	RecurrenceEnd  *time.Time     `json:"-"`
	QrToken        *string        `gorm:"uniqueIndex" json:"-"`
	AttendanceCode *string        `json:"-"`
	CodeExpiresAt  *time.Time     `json:"-"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	ClassRoom   Class        `gorm:"foreignKey:ClassID" json:"class,omitempty"`
	Course      Course       `gorm:"foreignKey:CourseID" json:"course,omitempty"`
	Location    Location     `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	Attendances []Attendance `json:"-"`
}

func (Schedule) TableName() string { return "schedules" }

const ScheduleTolerance = 5 * time.Minute

// RepeatDaySet mengurai CSV repeat_days menjadi himpunan weekday
// (time.Sunday=0 .. time.Saturday=6).
func (s *Schedule) RepeatDaySet() map[time.Weekday]bool {
	set := make(map[time.Weekday]bool)
	if s.RepeatDays == nil {
		return set
	}
	for _, part := range strings.Split(*s.RepeatDays, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && n >= 0 && n <= 6 {
			set[time.Weekday(n)] = true
		}
	}
	return set
}

// RepeatDaySlice mengekspos repeat_days sebagai []int terurut untuk API.
func (s *Schedule) RepeatDaySlice() []int {
	set := s.RepeatDaySet()
	days := make([]int, 0, len(set))
	for d := time.Sunday; d <= time.Saturday; d++ {
		if set[d] {
			days = append(days, int(d))
		}
	}
	return days
}

func dateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// OccursOn melaporkan apakah seri berulang aktif pada tanggal day
// (hari cocok, tidak sebelum occurrence pertama, dan belum lewat recurrence_end).
func (s *Schedule) OccursOn(day time.Time) bool {
	if !s.IsRecurring || s.RepeatDays == nil {
		return false
	}
	if dateOf(day).Before(dateOf(s.StartTime)) {
		return false
	}
	if s.RecurrenceEnd != nil && dateOf(day).After(dateOf(*s.RecurrenceEnd)) {
		return false
	}
	return s.RepeatDaySet()[day.Weekday()]
}

// EffectiveWindow memetakan jam mulai/selesai seri ke tanggal day
// (end sebelum start dianggap melewati tengah malam → end +24 jam).
func (s *Schedule) EffectiveWindow(day time.Time) (time.Time, time.Time) {
	loc := day.Location()
	start := time.Date(day.Year(), day.Month(), day.Day(),
		s.StartTime.Hour(), s.StartTime.Minute(), s.StartTime.Second(), 0, loc)
	end := time.Date(day.Year(), day.Month(), day.Day(),
		s.EndTime.Hour(), s.EndTime.Minute(), s.EndTime.Second(), 0, loc)
	if end.Before(start) {
		end = end.Add(24 * time.Hour)
	}
	return start, end
}

func (s *Schedule) IsActive(now time.Time) bool {
	if !s.IsRecurring {
		return !now.Before(s.StartTime.Add(-ScheduleTolerance)) && !now.After(s.EndTime.Add(ScheduleTolerance))
	}
	if !s.OccursOn(now) {
		return false
	}
	start, end := s.EffectiveWindow(now)
	return !now.Before(start.Add(-ScheduleTolerance)) && !now.After(end.Add(ScheduleTolerance))
}

// ActiveEndTime mengembalikan batas akhir window aktif pada tanggal now —
// dipakai untuk penentuan status terlambat pada jadwal recurring.
func (s *Schedule) ActiveEndTime(now time.Time) time.Time {
	if !s.IsRecurring {
		return s.EndTime
	}
	_, end := s.EffectiveWindow(now)
	return end
}

func (s *Schedule) IsCodeValid(now time.Time) bool {
	if s.AttendanceCode == nil || *s.AttendanceCode == "" || s.CodeExpiresAt == nil {
		return false
	}
	return !now.After(*s.CodeExpiresAt)
}
