package seeders

import (
	"errors"
	"fmt"
	"time"

	"github.com/edoazn/absensi-go/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB, bcryptCost int, tz *time.Location) error {
	pass, err := hashPassword("password", bcryptCost)
	if err != nil {
		return err
	}

	if _, err := ensureUser(db, "12345678", "Admin Kampus", "admin@kampus.ac.id", pass, models.RoleAdmin); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	budi, err := ensureUser(db, "211420108", "Budi Santoso", "budi@mahasiswa.ac.id", pass, models.RoleMahasiswa)
	if err != nil {
		return fmt.Errorf("seed budi: %w", err)
	}
	siti, err := ensureUser(db, "211420109", "Siti Rahayu", "siti@mahasiswa.ac.id", pass, models.RoleMahasiswa)
	if err != nil {
		return fmt.Errorf("seed siti: %w", err)
	}

	gedungA, err := ensureLocation(db, "Gedung A - Fakultas Teknik", -6.200000, 106.816666, 100)
	if err != nil {
		return fmt.Errorf("seed gedung A: %w", err)
	}
	gedungB, err := ensureLocation(db, "Gedung B - Fakultas Ekonomi", -6.201500, 106.818000, 150)
	if err != nil {
		return fmt.Errorf("seed gedung B: %w", err)
	}

	courses := map[string]models.Course{}
	for _, spec := range []struct{ code, name, lecturer string }{
		{"IF101", "Pemrograman Web", "Dr. Ahmad Fauzi"},
		{"IF102", "Basis Data", "Prof. Siti Nurhaliza"},
		{"EK201", "Ekonomi Digital", "Dr. Rina Marlina"},
	} {
		course, courseErr := ensureCourse(db, spec.code, spec.name, spec.lecturer)
		if courseErr != nil {
			return fmt.Errorf("seed course %s: %w", spec.code, courseErr)
		}
		courses[spec.code] = course
	}

	ti2a, err := ensureClass(db, "TI-2A", "2024/2025")
	if err != nil {
		return fmt.Errorf("seed TI-2A: %w", err)
	}
	ti2b, err := ensureClass(db, "TI-2B", "2024/2025")
	if err != nil {
		return fmt.Errorf("seed TI-2B: %w", err)
	}

	for _, student := range []models.User{budi, siti} {
		if !enrolled(db, ti2a.ID, student.ID) {
			if err := db.Model(&ti2a).Association("Students").Append(&student); err != nil {
				return fmt.Errorf("attach %s to TI-2A: %w", student.Name, err)
			}
		}
	}

	today := time.Now().In(tz)
	dayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, tz)

	demo01 := "DEMO01"
	schedules := []struct {
		classID uint
		course  models.Course
		loc     models.Location
		start   time.Time
		end     time.Time
		qr      bool
	}{
		{ti2a.ID, courses["IF101"], gedungA, dayStart.Add(8 * time.Hour), dayStart.Add(10 * time.Hour), false},
		{ti2a.ID, courses["IF102"], gedungB, dayStart.Add(10*time.Hour + 30*time.Minute), dayStart.Add(12*time.Hour + 30*time.Minute), false},
		{ti2b.ID, courses["EK201"], gedungA, dayStart.Add(13 * time.Hour), dayStart.Add(15 * time.Hour), false},
		{ti2a.ID, courses["IF101"], gedungA, dayStart, dayStart.Add(23*time.Hour + 59*time.Minute), true},
	}

	for _, sp := range schedules {
		var count int64
		if err := db.Model(&models.Schedule{}).
			Where("class_id = ? AND course_id = ? AND start_time = ?", sp.classID, sp.course.ID, sp.start).
			Count(&count).Error; err != nil {
			return fmt.Errorf("check schedule %s: %w", sp.course.CourseCode, err)
		}
		if count > 0 {
			continue
		}

		row := models.Schedule{
			ClassID:    sp.classID,
			CourseID:   sp.course.ID,
			LocationID: sp.loc.ID,
			StartTime:  sp.start,
			EndTime:    sp.end,
		}
		if sp.qr {
			token := uuid.NewString()
			expiry := time.Now().Add(7 * 24 * time.Hour)
			row.QrToken = &token
			row.AttendanceCode = &demo01
			row.CodeExpiresAt = &expiry
		}
		if err := db.Create(&row).Error; err != nil {
			return fmt.Errorf("seed schedule %s: %w", sp.course.CourseCode, err)
		}
	}

	return nil
}

func ensureUser(db *gorm.DB, identityNumber, name, email, passwordHash, role string) (models.User, error) {
	var user models.User
	err := db.Where("identity_number = ?", identityNumber).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = models.User{
			Name:           name,
			IdentityNumber: identityNumber,
			Email:          &email,
			Password:       passwordHash,
			Role:           role,
		}
		return user, db.Create(&user).Error
	}
	return user, err
}

func ensureLocation(db *gorm.DB, name string, lat, lon, radius float64) (models.Location, error) {
	var loc models.Location
	err := db.Where("name = ?", name).First(&loc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		loc = models.Location{Name: name, Latitude: lat, Longitude: lon, Radius: radius}
		return loc, db.Create(&loc).Error
	}
	return loc, err
}

func ensureCourse(db *gorm.DB, code, name, lecturer string) (models.Course, error) {
	var course models.Course
	err := db.Where("course_code = ?", code).First(&course).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		course = models.Course{CourseName: name, CourseCode: code, LecturerName: lecturer}
		return course, db.Create(&course).Error
	}
	return course, err
}

func ensureClass(db *gorm.DB, name, academicYear string) (models.Class, error) {
	var class models.Class
	err := db.Where("name = ?", name).First(&class).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		class = models.Class{Name: name, AcademicYear: academicYear}
		return class, db.Create(&class).Error
	}
	return class, err
}

func enrolled(db *gorm.DB, classID, userID uint) bool {
	var count int64
	db.Table("class_user").
		Where("class_id = ? AND user_id = ?", classID, userID).
		Count(&count)
	return count > 0
}

func hashPassword(plain string, cost int) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}
