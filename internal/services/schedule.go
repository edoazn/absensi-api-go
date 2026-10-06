package services

import (
	"math/rand/v2"
	"time"

	"github.com/edoazn/absensi-go/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ScheduleService struct {
	db *gorm.DB
	tz *time.Location
}

func NewScheduleService(db *gorm.DB, tz *time.Location) *ScheduleService {
	return &ScheduleService{db: db, tz: tz}
}

func (s *ScheduleService) GenerateAttendanceCode(scheduleID uint, minutesValid int) (string, time.Time, error) {
	code := randomUppercaseCode(6)
	now := time.Now().In(s.tz)
	expiresAt := now.Add(time.Duration(minutesValid) * time.Minute)

	err := s.db.Model(&models.Schedule{}).
		Where("id = ?", scheduleID).
		Updates(map[string]any{
			"attendance_code": code,
			"code_expires_at": expiresAt,
		}).Error
	if err != nil {
		return "", time.Time{}, err
	}
	return code, expiresAt, nil
}

func (s *ScheduleService) RegenerateQrToken(scheduleID uint) (string, error) {
	token := uuid.NewString()
	err := s.db.Model(&models.Schedule{}).
		Where("id = ?", scheduleID).
		Update("qr_token", token).Error
	if err != nil {
		return "", err
	}
	return token, nil
}

const codeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomUppercaseCode(length int) string {
	buf := make([]byte, length)
	for i := range buf {
		buf[i] = codeCharset[rand.IntN(len(codeCharset))]
	}
	return string(buf)
}
