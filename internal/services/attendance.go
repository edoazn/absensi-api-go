package services

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/edoazn/absensi-go/internal/models"
	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrAlreadyAttended = errors.New("already attended")

const (
	// MaxGpsAccuracyMeters: akurasi GPS device (meter) di atas ini dianggap
	// tidak layak untuk validasi kehadiran (posisi bisa melompat ratusan meter).
	MaxGpsAccuracyMeters = 500.0
	// MaxTravelSpeedMPS ≈ 900 km/h — hanya teleportasi yang melewati batas ini.
	MaxTravelSpeedMPS = 250.0
	// velocityWindow: jarak dibandingkan dengan hadir terakhir dalam rentang ini.
	velocityWindow = 90 * time.Minute
)

type AttendanceInput struct {
	ScheduleID     uint
	Method         string
	Latitude       *float64
	Longitude      *float64
	Accuracy       *float64
	QrToken        string
	AttendanceCode string
}

type AttendanceResult struct {
	Success    bool
	Status     string
	Distance   *float64
	Method     string
	Message    string
	Attendance *models.Attendance
}

type AttendanceService struct {
	db  *gorm.DB
	geo *GeolocationService
	tz  *time.Location
}

func NewAttendanceService(db *gorm.DB, tz *time.Location) *AttendanceService {
	return &AttendanceService{db: db, geo: NewGeolocationService(), tz: tz}
}

func (s *AttendanceService) Process(user *models.User, in AttendanceInput) AttendanceResult {
	var schedule models.Schedule
	if err := s.db.Preload("Location").First(&schedule, in.ScheduleID).Error; err != nil {
		return s.fail("Jadwal tidak ditemukan")
	}

	if !s.enrolledInClass(user.ID, schedule.ClassID) {
		return s.fail("Anda tidak terdaftar di kelas ini")
	}

	now := time.Now().In(s.tz)
	if !schedule.IsActive(now) {
		return s.fail("Absensi hanya dapat dilakukan pada waktu jadwal aktif")
	}

	switch in.Method {
	case models.MethodGeolocation:
		return s.handleGeolocation(user, &schedule, in)
	case models.MethodQrCode:
		return s.handleQrCode(user, &schedule, strings.TrimSpace(in.QrToken), in)
	case models.MethodAttendanceCode:
		return s.handleAttendanceCode(user, &schedule, strings.ToUpper(strings.TrimSpace(in.AttendanceCode)), now)
	default:
		return s.fail("Metode absensi tidak valid")
	}
}

func (s *AttendanceService) handleGeolocation(user *models.User, schedule *models.Schedule, in AttendanceInput) AttendanceResult {
	distance := s.geo.CalculateDistance(*in.Latitude, *in.Longitude, schedule.Location.Latitude, schedule.Location.Longitude)
	rounded := math.Round(distance*100) / 100

	status, message := s.geoVerdict(user, schedule, in, time.Now().In(s.tz), rounded,
		"Absensi berhasil dicatat", "Absensi berhasil dicatat dengan status terlambat")

	attendance := &models.Attendance{
		UserID:      user.ID,
		ScheduleID:  schedule.ID,
		Latitude:    in.Latitude,
		Longitude:   in.Longitude,
		Distance:    &rounded,
		GpsAccuracy: in.Accuracy,
		Status:      status,
		Method:      models.MethodGeolocation,
	}
	if err := s.persist(attendance); err != nil {
		return s.failFromPersist(err)
	}

	return AttendanceResult{
		Success: true, Status: status, Distance: &rounded,
		Method: models.MethodGeolocation, Message: message, Attendance: attendance,
	}
}

// geoVerdict menentukan status akhir metode berbasis koordinat dengan urutan:
// akurasi GPS → radius → impossible travel → ketepatan waktu (hadir/terlambat).
func (s *AttendanceService) geoVerdict(user *models.User, schedule *models.Schedule, in AttendanceInput, now time.Time, distance float64, onTimeMsg, lateMsg string) (string, string) {
	if in.Accuracy != nil && *in.Accuracy > MaxGpsAccuracyMeters {
		return models.StatusDitolak, "Absensi ditolak karena akurasi GPS terlalu rendah untuk validasi kehadiran"
	}
	if distance > schedule.Location.Radius {
		return models.StatusDitolak, "Absensi ditolak karena lokasi di luar radius"
	}
	if s.impossibleTravel(user.ID, *in.Latitude, *in.Longitude, now) {
		return models.StatusDitolak, "Absensi ditolak karena perpindahan lokasi tidak wajar"
	}
	if now.After(schedule.ActiveEndTime(now)) {
		return models.StatusTerlambat, lateMsg
	}
	return models.StatusHadir, onTimeMsg
}

// impossibleTravel membandingkan koordinat yang diklaim sekarang dengan hadir
// terakhir (berkoordinat) dalam window velocityWindow: kecepatan implikasi di
// atas MaxTravelSpeedMPS berarti posisi dipalsukan/dilompatkan.
// Referensi hanya baris 'hadir' agar retry setelah ditolak tidak salah flag.
func (s *AttendanceService) impossibleTravel(userID uint, lat, lon float64, now time.Time) bool {
	var prev models.Attendance
	err := s.db.
		Where("user_id = ? AND status = ? AND latitude IS NOT NULL AND longitude IS NOT NULL AND created_at >= ?",
			userID, models.StatusHadir, now.Add(-velocityWindow)).
		Order("created_at DESC").
		First(&prev).Error
	if err != nil {
		return false
	}
	seconds := now.Sub(prev.CreatedAt).Seconds()
	if seconds < 1 {
		seconds = 1
	}
	travel := s.geo.CalculateDistance(lat, lon, *prev.Latitude, *prev.Longitude)
	return travel/seconds > MaxTravelSpeedMPS
}

// handleQrCode memvalidasi token QR lalu TETAP memverifikasi lokasi mahasiswa:
// QR yang dibagikan (screenshot) tidak berguna tanpa berada di radius lokasi.
func (s *AttendanceService) handleQrCode(user *models.User, schedule *models.Schedule, token string, in AttendanceInput) AttendanceResult {
	if schedule.QrToken == nil || token == "" || *schedule.QrToken != token {
		return s.fail("QR Code tidak valid untuk jadwal ini")
	}
	if in.Latitude == nil || in.Longitude == nil {
		return s.fail("Koordinat lokasi wajib disertakan untuk absensi via QR Code")
	}

	distance := s.geo.CalculateDistance(*in.Latitude, *in.Longitude, schedule.Location.Latitude, schedule.Location.Longitude)
	rounded := math.Round(distance*100) / 100

	status, message := s.geoVerdict(user, schedule, in, time.Now().In(s.tz), rounded,
		"Absensi via QR Code berhasil dicatat", "Absensi via QR Code tercatat dengan status terlambat")

	attendance := &models.Attendance{
		UserID:      user.ID,
		ScheduleID:  schedule.ID,
		Latitude:    in.Latitude,
		Longitude:   in.Longitude,
		Distance:    &rounded,
		GpsAccuracy: in.Accuracy,
		Status:      status,
		Method:      models.MethodQrCode,
	}
	if err := s.persist(attendance); err != nil {
		return s.failFromPersist(err)
	}

	return AttendanceResult{
		Success: true, Status: status, Distance: &rounded,
		Method: models.MethodQrCode, Message: message, Attendance: attendance,
	}
}

func (s *AttendanceService) handleAttendanceCode(user *models.User, schedule *models.Schedule, code string, now time.Time) AttendanceResult {
	if schedule.AttendanceCode == nil || *schedule.AttendanceCode == "" {
		return s.fail("Kode absensi belum dibuat untuk jadwal ini")
	}
	if *schedule.AttendanceCode != code {
		return s.fail("Kode absensi tidak valid")
	}
	if !schedule.IsCodeValid(now) {
		return s.fail("Kode absensi sudah kedaluwarsa")
	}

	status := models.StatusHadir
	message := "Absensi via kode manual berhasil dicatat"
	if now.After(schedule.ActiveEndTime(now)) {
		status = models.StatusTerlambat
		message = "Absensi via kode manual tercatat dengan status terlambat"
	}

	attendance := &models.Attendance{
		UserID:     user.ID,
		ScheduleID: schedule.ID,
		Status:     status,
		Method:     models.MethodAttendanceCode,
	}
	if err := s.persist(attendance); err != nil {
		return s.failFromPersist(err)
	}

	return AttendanceResult{
		Success: true, Status: status,
		Method: models.MethodAttendanceCode, Message: message, Attendance: attendance,
	}
}

func (s *AttendanceService) persist(att *models.Attendance) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var locked models.Schedule
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").First(&locked, att.ScheduleID).Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&models.Attendance{}).
			Where("user_id = ? AND schedule_id = ? AND status IN ?", att.UserID, att.ScheduleID, []string{models.StatusHadir, models.StatusTerlambat}).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrAlreadyAttended
		}

		return tx.Create(att).Error
	})
	if errors.Is(err, ErrAlreadyAttended) {
		return err
	}
	if isDuplicateEntry(err) {
		return ErrAlreadyAttended
	}
	return err
}

func (s *AttendanceService) fail(message string) AttendanceResult {
	return AttendanceResult{Success: false, Message: message}
}

func (s *AttendanceService) failFromPersist(err error) AttendanceResult {
	if errors.Is(err, ErrAlreadyAttended) {
		return s.fail("Anda sudah melakukan absensi untuk jadwal ini")
	}
	return s.fail("Gagal mencatat absensi")
}

func (s *AttendanceService) enrolledInClass(userID, classID uint) bool {
	var count int64
	s.db.Table("class_user").
		Where("class_id = ? AND user_id = ?", classID, userID).
		Count(&count)
	return count > 0
}

func isDuplicateEntry(err error) bool {
	var mysqlErr *gomysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func (s *AttendanceService) TodaySchedules(user *models.User) ([]models.Schedule, error) {
	classIDs := []uint{}
	if err := s.db.Table("class_user").
		Where("user_id = ?", user.ID).
		Pluck("class_id", &classIDs).Error; err != nil {
		return nil, err
	}

	today := time.Now().In(s.tz)
	dayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, s.tz)
	dayEnd := dayStart.Add(24 * time.Hour)

	preloads := func(db *gorm.DB) *gorm.DB {
		return db.
			Preload("ClassRoom", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
			Preload("Course", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
			Preload("Location", func(db *gorm.DB) *gorm.DB { return db.Unscoped() })
	}

	var schedules []models.Schedule
	// Jadwal sekali-jalan: occurrence-nya memang jatuh hari ini.
	if err := preloads(s.db).
		Where("class_id IN ? AND is_recurring = ?", classIDs, false).
		Where("start_time >= ? AND start_time < ?", dayStart, dayEnd).
		Find(&schedules).Error; err != nil {
		return nil, err
	}

	// Seri mingguan: ambil kandidat lalu hitung window efektif hari ini.
	var series []models.Schedule
	if err := preloads(s.db).
		Where("class_id IN ? AND is_recurring = ?", classIDs, true).
		Where("start_time < ?", dayEnd).
		Where("recurrence_end IS NULL OR recurrence_end >= ?", dayStart).
		Find(&series).Error; err != nil {
		return nil, err
	}
	for _, sch := range series {
		if !sch.OccursOn(today) {
			continue
		}
		start, end := sch.EffectiveWindow(today)
		sch.StartTime = start
		sch.EndTime = end
		schedules = append(schedules, sch)
	}

	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].StartTime.Before(schedules[j].StartTime)
	})
	return schedules, nil
}

func (s *AttendanceService) History(userID uint, page, perPage int) ([]models.Attendance, int64, error) {
	query := s.db.Model(&models.Attendance{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	var rows []models.Attendance
	err := query.
		Preload("Schedule", func(db *gorm.DB) *gorm.DB { return db.Unscoped().Preload("Course") }).
		Order("created_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&rows).Error
	return rows, total, err
}
