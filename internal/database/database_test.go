package database_test

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/edoazn/absensi-go/config"
	"github.com/edoazn/absensi-go/internal/database"
	"github.com/edoazn/absensi-go/internal/models"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	return &config.Config{
		DBUser:     "root",
		DBPassword: "root",
		DBHost:     "127.0.0.1",
		DBPort:     "3306",
		DBName:     fmt.Sprintf("absensi_go_test_%d", os.Getpid()),
		Timezone:   loc,
	}
}

func setupTestDB(t *testing.T) (*config.Config, *gorm.DB) {
	t.Helper()
	cfg := testConfig(t)

	if err := database.EnsureDatabase(cfg); err != nil {
		t.Skipf("mysql not available, skipping integration test: %v", err)
	}
	if err := database.Migrate(cfg); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db, err := database.Connect(cfg)
	if err != nil {
		t.Skipf("mysql not available, skipping integration test: %v", err)
	}
	truncateAll(t, db)
	return cfg, db
}

func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"attendances", "refresh_tokens", "schedules", "courses",
		"locations", "class_user", "classes", "users",
	}
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		t.Fatalf("disable FK checks: %v", err)
	}
	for _, tb := range tables {
		if err := db.Exec("TRUNCATE TABLE " + tb).Error; err != nil {
			t.Fatalf("truncate %s: %v", tb, err)
		}
	}
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
		t.Fatalf("enable FK checks: %v", err)
	}
}

type fixture struct {
	user     models.User
	partner  models.User
	class    models.Class
	course   models.Course
	location models.Location
	schedule models.Schedule
}

func createFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()
	loc, _ := time.LoadLocation("Asia/Jakarta")

	user := models.User{
		Name:           "Budi Santoso",
		IdentityNumber: "211420108",
		Email:          ptrString("budi@test.ac.id"),
		Password:       "hashed",
		Role:           models.RoleMahasiswa,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	partner := models.User{
		Name:           "Siti Rahayu",
		IdentityNumber: "211420109",
		Email:          ptrString("siti@test.ac.id"),
		Password:       "hashed",
		Role:           models.RoleMahasiswa,
	}
	if err := db.Create(&partner).Error; err != nil {
		t.Fatalf("create partner: %v", err)
	}

	class := models.Class{Name: "TI-2A", AcademicYear: "2024/2025"}
	if err := db.Create(&class).Error; err != nil {
		t.Fatalf("create class: %v", err)
	}
	if err := db.Model(&class).Association("Students").Append(&user, &partner); err != nil {
		t.Fatalf("enroll students: %v", err)
	}

	course := models.Course{CourseName: "Pemrograman Web", CourseCode: "IF101", LecturerName: "Dr. A"}
	if err := db.Create(&course).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}

	location := models.Location{Name: "Gedung A", Latitude: -6.2, Longitude: 106.816666, Radius: 100}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("create location: %v", err)
	}

	day := time.Date(2026, 8, 22, 0, 0, 0, 0, loc)
	schedule := models.Schedule{
		ClassID:    class.ID,
		CourseID:   course.ID,
		LocationID: location.ID,
		StartTime:  day.Add(10 * time.Hour),
		EndTime:    day.Add(12 * time.Hour),
	}
	if err := db.Create(&schedule).Error; err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	return fixture{user: user, partner: partner, class: class, course: course, location: location, schedule: schedule}
}

func ptrString(s string) *string  { return &s }
func ptrFloat(f float64) *float64 { return &f }

func isDuplicateEntry(err error) bool {
	var mysqlErr *gomysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func TestMigrateIsIdempotent(t *testing.T) {
	cfg := testConfig(t)
	if err := database.EnsureDatabase(cfg); err != nil {
		t.Skipf("mysql not available: %v", err)
	}
	if err := database.Migrate(cfg); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := database.Migrate(cfg); err != nil {
		t.Fatalf("second migrate must be a no-op, got: %v", err)
	}
}

func TestHadirDuplicateBlockedByDBConstraint(t *testing.T) {
	_, db := setupTestDB(t)
	f := createFixture(t, db)

	first := models.Attendance{
		UserID: f.user.ID, ScheduleID: f.schedule.ID,
		Latitude: ptrFloat(-6.2), Longitude: ptrFloat(106.816666), Distance: ptrFloat(50),
		Status: models.StatusHadir, Method: models.MethodGeolocation,
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("first hadir must succeed: %v", err)
	}

	dup := models.Attendance{
		UserID: f.user.ID, ScheduleID: f.schedule.ID,
		Latitude: ptrFloat(-6.2), Longitude: ptrFloat(106.816666), Distance: ptrFloat(80),
		Status: models.StatusHadir, Method: models.MethodGeolocation,
	}
	err := db.Create(&dup).Error
	if err == nil {
		t.Fatal("second hadir must be rejected by unique constraint")
	}
	if !isDuplicateEntry(err) {
		t.Fatalf("expected duplicate entry error 1062, got: %v", err)
	}
}

func TestDitolakRetryAllowed(t *testing.T) {
	_, db := setupTestDB(t)
	f := createFixture(t, db)

	first := models.Attendance{
		UserID: f.user.ID, ScheduleID: f.schedule.ID,
		Latitude: ptrFloat(-6.3), Longitude: ptrFloat(106.9), Distance: ptrFloat(5000),
		Status: models.StatusDitolak, Method: models.MethodGeolocation,
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("first ditolak must succeed: %v", err)
	}

	retry := models.Attendance{
		UserID: f.user.ID, ScheduleID: f.schedule.ID,
		Latitude: ptrFloat(-6.2), Longitude: ptrFloat(106.816666), Distance: ptrFloat(40),
		Status: models.StatusHadir, Method: models.MethodGeolocation,
	}
	if err := db.Create(&retry).Error; err != nil {
		t.Fatalf("retry after ditolak must succeed: %v", err)
	}

	var count int64
	db.Model(&models.Attendance{}).Where("user_id = ?", f.user.ID).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 attendance rows, got %d", count)
	}
}

func TestMultipleDitolakRowsAllowed(t *testing.T) {
	_, db := setupTestDB(t)
	f := createFixture(t, db)

	for i := range 3 {
		row := models.Attendance{
			UserID: f.user.ID, ScheduleID: f.schedule.ID,
			Latitude: ptrFloat(-6.3), Longitude: ptrFloat(106.9), Distance: ptrFloat(float64(5000 + i)),
			Status: models.StatusDitolak, Method: models.MethodGeolocation,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("ditolak row %d must succeed: %v", i+1, err)
		}
	}
}

func TestSoftDeletedHadirFreesSlot(t *testing.T) {
	_, db := setupTestDB(t)
	f := createFixture(t, db)

	first := models.Attendance{
		UserID: f.user.ID, ScheduleID: f.schedule.ID,
		Latitude: ptrFloat(-6.2), Longitude: ptrFloat(106.816666), Distance: ptrFloat(50),
		Status: models.StatusHadir, Method: models.MethodGeolocation,
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create hadir: %v", err)
	}
	if err := db.Delete(&first).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	again := models.Attendance{
		UserID: f.user.ID, ScheduleID: f.schedule.ID,
		Latitude: ptrFloat(-6.2), Longitude: ptrFloat(106.816666), Distance: ptrFloat(60),
		Status: models.StatusHadir, Method: models.MethodQrCode,
	}
	if err := db.Create(&again).Error; err != nil {
		t.Fatalf("hadir after soft delete must succeed: %v", err)
	}
}

func TestDifferentUsersSameScheduleHadir(t *testing.T) {
	_, db := setupTestDB(t)
	f := createFixture(t, db)

	for _, uid := range []uint{f.user.ID, f.partner.ID} {
		row := models.Attendance{
			UserID: uid, ScheduleID: f.schedule.ID,
			Status: models.StatusHadir, Method: models.MethodAttendanceCode,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("different users must both be able to hadir: %v", err)
		}
	}
}

func TestScheduleIsActiveTolerance(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	day := time.Date(2026, 8, 22, 0, 0, 0, 0, loc)

	s := models.Schedule{
		StartTime: day.Add(10 * time.Hour),
		EndTime:   day.Add(12 * time.Hour),
	}

	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"before window", day.Add(9*time.Hour + 54*time.Minute), false},
		{"exactly tolerance edge", day.Add(9*time.Hour + 55*time.Minute), true},
		{"inside", day.Add(11 * time.Hour), true},
		{"end tolerance edge", day.Add(12*time.Hour + 5*time.Minute), true},
		{"after tolerance", day.Add(12*time.Hour + 6*time.Minute), false},
	}

	for _, tc := range cases {
		if got := s.IsActive(tc.at); got != tc.want {
			t.Errorf("%s: IsActive=%v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestScheduleIsCodeValid(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Date(2026, 8, 22, 10, 0, 0, 0, loc)
	expired := now.Add(-time.Hour)
	valid := now.Add(time.Hour)

	code := "DEMO01"
	if (&models.Schedule{}).IsCodeValid(now) {
		t.Error("empty code must be invalid")
	}
	if (&models.Schedule{AttendanceCode: &code}).IsCodeValid(now) {
		t.Error("code without expiry must be invalid")
	}
	if !(&models.Schedule{AttendanceCode: &code, CodeExpiresAt: &valid}).IsCodeValid(now) {
		t.Error("valid code must pass")
	}
	if (&models.Schedule{AttendanceCode: &code, CodeExpiresAt: &expired}).IsCodeValid(now) {
		t.Error("expired code must fail")
	}
}
