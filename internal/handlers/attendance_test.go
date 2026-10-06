package handlers_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/edoazn/absensi-go/config"
	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/edoazn/absensi-go/internal/server"
	"github.com/edoazn/absensi-go/internal/testsupport"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func buildRouterForTest(cfg *config.Config, db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return server.BuildRouter(server.Deps{Cfg: cfg, DB: db})
}

type attFix struct {
	db               *gorm.DB
	r                *gin.Engine
	tz               *time.Location
	student          models.User
	outsider         models.User
	class            models.Class
	loc              models.Location
	active           models.Schedule
	qrSched          models.Schedule
	codeSched        models.Schedule
	expiredCodeSched models.Schedule
	past             models.Schedule
}

const qrToken = "11111111-2222-3333-4444-555555555555"

func makeAttFix(t *testing.T) attFix {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := testsupport.TestConfig(t)
	_, db := testsupport.SetupDB(t)
	r := buildRouterForTest(cfg, db)
	f := attFix{db: db, r: r, tz: cfg.Timezone}

	now := time.Now().In(f.tz)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, f.tz)

	f.class = models.Class{Name: "TI-3A", AcademicYear: "2025/2026"}
	if err := db.Create(&f.class).Error; err != nil {
		t.Fatalf("create class: %v", err)
	}
	course := models.Course{CourseName: "Go Programming", CourseCode: "GO301", LecturerName: "Pak Gopher"}
	if err := db.Create(&course).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	otherCourse := models.Course{CourseName: "Other", CourseCode: "OT999", LecturerName: "X"}
	db.Create(&otherCourse)
	f.loc = models.Location{Name: "Lab PPLS", Latitude: -6.2, Longitude: 106.816666, Radius: 100}
	if err := db.Create(&f.loc).Error; err != nil {
		t.Fatalf("create location: %v", err)
	}
	otherLoc := models.Location{Name: "Ruang Lain", Latitude: -6.3, Longitude: 106.9, Radius: 50}
	db.Create(&otherLoc)

	f.student = testsupport.CreateUser(t, db, "Mahasiswa Satu", "301220001", models.RoleMahasiswa, nil)
	f.outsider = testsupport.CreateUser(t, db, "Mahasiswa Dua", "301220002", models.RoleMahasiswa, nil)
	testsupport.CreateUser(t, db, "Admin Uji", "00000001", models.RoleAdmin, nil)

	if err := db.Model(&f.class).Association("Students").Append(&f.student); err != nil {
		t.Fatalf("enroll student: %v", err)
	}
	otherClass := models.Class{Name: "TI-3B", AcademicYear: "2025/2026"}
	db.Create(&otherClass)

	mk := func(classID uint, start, end time.Time) models.Schedule {
		s := models.Schedule{ClassID: classID, CourseID: course.ID, LocationID: f.loc.ID, StartTime: start, EndTime: end}
		if err := db.Create(&s).Error; err != nil {
			t.Fatalf("create schedule: %v", err)
		}
		return s
	}

	f.active = mk(f.class.ID, now.Add(-30*time.Minute), now.Add(60*time.Minute))
	f.past = mk(f.class.ID, now.Add(-48*time.Hour), now.Add(-47*time.Hour))
	mk(f.class.ID, now.Add(24*time.Hour), now.Add(25*time.Hour))
	mk(otherClass.ID, dayStart.Add(5*time.Hour), dayStart.Add(7*time.Hour))

	f.qrSched = mk(f.class.ID, now.Add(-15*time.Minute), now.Add(45*time.Minute))
	token := qrToken
	f.qrSched.QrToken = &token
	db.Save(&f.qrSched)

	demo := "DEMO01"
	exp := now.Add(30 * time.Minute)
	f.codeSched = mk(f.class.ID, now.Add(-15*time.Minute), now.Add(45*time.Minute))
	f.codeSched.AttendanceCode = &demo
	f.codeSched.CodeExpiresAt = &exp
	db.Save(&f.codeSched)

	old := "OLD999"
	oldExp := now.Add(-time.Hour)
	f.expiredCodeSched = mk(f.class.ID, now.Add(-15*time.Minute), now.Add(45*time.Minute))
	f.expiredCodeSched.AttendanceCode = &old
	f.expiredCodeSched.CodeExpiresAt = &oldExp
	db.Save(&f.expiredCodeSched)

	return f
}

func bearerFor(t *testing.T, f attFix, user models.User) string {
	t.Helper()
	status, data, env := loginRequest(t, f.r, user.IdentityNumber, testsupport.TestPassword)
	if status != http.StatusOK {
		t.Fatalf("login %s failed (%d): %+v", user.IdentityNumber, status, env)
	}
	return data.AccessToken
}

type attStoreData struct {
	Status     string   `json:"status"`
	Distance   *float64 `json:"distance"`
	Method     string   `json:"method"`
	Message    string   `json:"message"`
	Attendance struct {
		ID     uint   `json:"id"`
		Status string `json:"status"`
		Method string `json:"method"`
	} `json:"attendance"`
}

func submitAttendance(t *testing.T, f attFix, bearer string, body map[string]any) (int, api.Envelope, attStoreData) {
	t.Helper()
	status, env := doJSON(t, f.r, http.MethodPost, "/api/v1/attendance", bearer, body)
	var data attStoreData
	raw, _ := json.Marshal(env.Data)
	_ = json.Unmarshal(raw, &data)
	return status, env, data
}

func geoBody(scheduleID uint, lat, lon float64) map[string]any {
	return map[string]any{
		"schedule_id": scheduleID,
		"method":      "geolocation",
		"latitude":    lat,
		"longitude":   lon,
	}
}

func TestGeolocationHadirWithinRadius(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	status, env, data := submitAttendance(t, f, bearer, geoBody(f.active.ID, -6.20005, 106.816666))

	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %+v", status, env)
	}
	if data.Status != models.StatusHadir || data.Method != models.MethodGeolocation {
		t.Fatalf("unexpected result: %+v", data)
	}
	if data.Distance == nil || *data.Distance > 100 {
		t.Fatalf("distance must be present and <= radius: %v", data.Distance)
	}
	if data.Message != "Absensi berhasil dicatat" {
		t.Fatalf("message = %q", data.Message)
	}

	var count int64
	f.db.Model(&models.Attendance{}).Where("schedule_id = ?", f.active.ID).Count(&count)
	if count != 1 {
		t.Fatalf("attendance rows = %d, want 1", count)
	}
}

func TestGeolocationDitolakOutsideRadiusThenRetryHadir(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	status, _, data := submitAttendance(t, f, bearer, geoBody(f.active.ID, -6.21, 106.83))

	if status != http.StatusOK {
		t.Fatalf("ditolak status = %d, want 200", status)
	}
	if data.Status != models.StatusDitolak {
		t.Fatalf("status = %q, want ditolak", data.Status)
	}
	if data.Message != "Absensi ditolak karena lokasi di luar radius" {
		t.Fatalf("message = %q", data.Message)
	}

	retryStatus, _, retryData := submitAttendance(t, f, bearer, geoBody(f.active.ID, -6.20005, 106.816666))
	if retryStatus != http.StatusCreated || retryData.Status != models.StatusHadir {
		t.Fatalf("retry after ditolak must succeed: %d %+v", retryStatus, retryData)
	}

	var count int64
	f.db.Model(&models.Attendance{}).Where("user_id = ? AND schedule_id = ?", f.student.ID, f.active.ID).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 rows (ditolak+hadir), got %d", count)
	}
}

func TestDuplicateHadirBlocked(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	if st, _, d := submitAttendance(t, f, bearer, geoBody(f.active.ID, -6.20005, 106.816666)); st != http.StatusCreated || d.Status != models.StatusHadir {
		t.Fatalf("first submission setup failed: %d %+v", st, d)
	}

	status, env, _ := submitAttendance(t, f, bearer, geoBody(f.active.ID, -6.2001, 106.8167))

	if status != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate must be 422, got %d", status)
	}
	if env.Message != "Anda sudah melakukan absensi untuk jadwal ini" {
		t.Fatalf("message = %q", env.Message)
	}
}

func TestNotEnrolledClassRejected(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.outsider)

	status, env, _ := submitAttendance(t, f, bearer, geoBody(f.active.ID, -6.20005, 106.816666))

	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", status)
	}
	if env.Message != "Anda tidak terdaftar di kelas ini" {
		t.Fatalf("message = %q", env.Message)
	}
}

func TestInactiveScheduleRejected(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	status, env, _ := submitAttendance(t, f, bearer, geoBody(f.past.ID, -6.20005, 106.816666))

	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", status)
	}
	if env.Message != "Absensi hanya dapat dilakukan pada waktu jadwal aktif" {
		t.Fatalf("message = %q", env.Message)
	}
}

func TestUnknownScheduleRejected(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	status, env, _ := submitAttendance(t, f, bearer, geoBody(99999, -6.20005, 106.816666))

	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", status)
	}
	if env.Message != "Jadwal tidak ditemukan" {
		t.Fatalf("message = %q", env.Message)
	}
}

func TestQrCodeFlow(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	status, _, data := submitAttendance(t, f, bearer, map[string]any{
		"schedule_id": f.qrSched.ID,
		"method":      "qr_code",
		"qr_token":    qrToken,
		"latitude":    -6.20005,
		"longitude":   106.816666,
	})
	if status != http.StatusCreated || data.Status != models.StatusHadir || data.Method != models.MethodQrCode {
		t.Fatalf("valid QR failed: %d %+v", status, data)
	}
	if data.Distance == nil || *data.Distance > 100 {
		t.Fatal("qr_code must carry distance within radius")
	}

	wrongStatus, wrongEnv, _ := submitAttendance(t, f, bearer, map[string]any{
		"schedule_id": f.qrSched.ID,
		"method":      "qr_code",
		"qr_token":    "token-salah",
		"latitude":    -6.20005,
		"longitude":   106.816666,
	})
	if wrongStatus != http.StatusUnprocessableEntity || wrongEnv.Message != "QR Code tidak valid untuk jadwal ini" {
		t.Fatalf("invalid QR: %d %+v", wrongStatus, wrongEnv)
	}
}

func TestQrCodeRequiresCoordinates(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	status, env, _ := submitAttendance(t, f, bearer, map[string]any{
		"schedule_id": f.qrSched.ID,
		"method":      "qr_code",
		"qr_token":    qrToken,
	})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("QR without coords must 422, got %d", status)
	}
	if !strings.Contains(env.Message, "Koordinat") {
		t.Fatalf("message = %q", env.Message)
	}
}

func TestQrCodeOutsideRadiusRejected(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	status, _, data := submitAttendance(t, f, bearer, map[string]any{
		"schedule_id": f.qrSched.ID,
		"method":      "qr_code",
		"qr_token":    qrToken,
		"latitude":    -6.21,
		"longitude":   106.83,
	})
	if status != http.StatusOK || data.Status != models.StatusDitolak || data.Distance == nil {
		t.Fatalf("out-of-radius QR = %d %+v", status, data)
	}

	var row models.Attendance
	if err := f.db.Where("user_id = ? AND schedule_id = ?", f.student.ID, f.qrSched.ID).First(&row).Error; err != nil {
		t.Fatalf("ditolak row missing: %v", err)
	}
	if row.Status != models.StatusDitolak || row.Latitude == nil || row.Distance == nil {
		t.Fatalf("row wrong: %+v", row)
	}
}

func TestAttendanceCodeFlows(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	status, env, _ := submitAttendance(t, f, bearer, map[string]any{
		"schedule_id":     f.active.ID,
		"method":          "attendance_code",
		"attendance_code": "DEMO01",
	})
	if status != http.StatusUnprocessableEntity || env.Message != "Kode absensi belum dibuat untuk jadwal ini" {
		t.Fatalf("no-code case: %d %+v", status, env)
	}

	status, env, _ = submitAttendance(t, f, bearer, map[string]any{
		"schedule_id":     f.codeSched.ID,
		"method":          "attendance_code",
		"attendance_code": "WRONGX",
	})
	if status != http.StatusUnprocessableEntity || env.Message != "Kode absensi tidak valid" {
		t.Fatalf("wrong-code case: %d %+v", status, env)
	}

	status, env, _ = submitAttendance(t, f, bearer, map[string]any{
		"schedule_id":     f.expiredCodeSched.ID,
		"method":          "attendance_code",
		"attendance_code": "OLD999",
	})
	if status != http.StatusUnprocessableEntity || env.Message != "Kode absensi sudah kedaluwarsa" {
		t.Fatalf("expired case: %d %+v", status, env)
	}

	status, _, data := submitAttendance(t, f, bearer, map[string]any{
		"schedule_id":     f.codeSched.ID,
		"method":          "attendance_code",
		"attendance_code": "demo01",
	})
	if status != http.StatusCreated || data.Status != models.StatusHadir || data.Method != models.MethodAttendanceCode {
		t.Fatalf("valid code (lowercase) must pass: %d %+v", status, data)
	}
}

func TestHistoryScopedOrderedPaginated(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	base := time.Now().In(f.tz).Add(-4 * time.Hour)
	for i := range 3 {
		sched := models.Schedule{
			ClassID: f.class.ID, CourseID: f.active.CourseID, LocationID: f.loc.ID,
			StartTime: base.Add(time.Duration(i) * time.Hour),
			EndTime:   base.Add(time.Duration(i)*time.Hour + 2*time.Hour),
		}
		if err := f.db.Create(&sched).Error; err != nil {
			t.Fatalf("seed history schedule: %v", err)
		}
		row := models.Attendance{
			UserID: f.student.ID, ScheduleID: sched.ID,
			Status: models.StatusHadir, Method: models.MethodQrCode,
			CreatedAt: base.Add(time.Duration(i) * time.Hour),
		}
		if err := f.db.Create(&row).Error; err != nil {
			t.Fatalf("seed history: %v", err)
		}
	}
	otherRow := models.Attendance{
		UserID: f.outsider.ID, ScheduleID: f.active.ID,
		Status: models.StatusHadir, Method: models.MethodQrCode,
		CreatedAt: base.Add(2 * time.Hour),
	}
	f.db.Create(&otherRow)

	status, env := doJSON(t, f.r, http.MethodGet, "/api/v1/attendance/history?page=1&per_page=2", bearer, nil)
	if status != http.StatusOK {
		t.Fatalf("history status = %d; body %+v", status, env)
	}

	raw, _ := json.Marshal(env.Data)
	var payload struct {
		Items []struct {
			ID        uint   `json:"id"`
			Status    string `json:"status"`
			CreatedAt string `json:"created_at"`
		} `json:"items"`
		Meta struct {
			CurrentPage int   `json:"current_page"`
			PerPage     int   `json:"per_page"`
			Total       int64 `json:"total"`
			LastPage    int   `json:"last_page"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode history: %v — %s", err, raw)
	}

	if payload.Meta.Total != 3 || payload.Meta.LastPage != 2 {
		t.Fatalf("meta mismatch: %+v (must exclude other users)", payload.Meta)
	}
	if len(payload.Items) != 2 {
		t.Fatalf("page size mismatch: %d", len(payload.Items))
	}
	first, second := payload.Items[0].CreatedAt, payload.Items[1].CreatedAt
	if first <= second {
		t.Fatalf("history not descending: %q then %q", first, second)
	}
}

func TestTodaySchedulesOnlyOwnClassesToday(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	status, env := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/today", bearer, nil)
	if status != http.StatusOK {
		t.Fatalf("today status = %d; body %+v", status, env)
	}

	raw, _ := json.Marshal(env.Data)
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatalf("decode schedules: %v", err)
	}

	if len(items) < 3 {
		t.Fatalf("expected >=3 own-class today schedules (active/past/future excluded by date only), got %d: %s", len(items), raw)
	}
	prev := ""
	for _, item := range items {
		start, _ := item["start_time"].(string)
		if prev != "" && start < prev {
			t.Fatalf("schedules not ascending: %q after %q", start, prev)
		}
		prev = start
	}

	outBearer := bearerFor(t, f, f.outsider)
	outStatus, outEnv := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/today", outBearer, nil)
	rawOut, _ := json.Marshal(outEnv.Data)
	var outItems []map[string]any
	json.Unmarshal(rawOut, &outItems)
	if outStatus != http.StatusOK || len(outItems) != 0 {
		t.Fatalf("outsider must see zero schedules, got %d", len(outItems))
	}
}

func TestRateLimitOnAttendanceEndpoint(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	last := 0
	for i := range 11 {
		status, _, _ := submitAttendance(t, f, bearer, geoBody(f.active.ID, -6.20005, 106.816666))
		last = status
		if i == 9 && status == http.StatusTooManyRequests {
			t.Fatal("request #10 must still be allowed")
		}
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("request #11 must be rate limited (429), got %d", last)
	}
}
