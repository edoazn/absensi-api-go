package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/gin-gonic/gin"
)

// doJSONFrom sama dengan doJSON tetapi RemoteAddr bisa diatur —
// dipakai untuk membuktikan rate limit per-user (bukan per-IP).
func doJSONFrom(r *gin.Engine, method, path, bearer string, body any, remoteAddr string) (int, api.Envelope) {
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.RemoteAddr = remoteAddr
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var env api.Envelope
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	return rec.Code, env
}

func TestUserCannotDeleteSelf(t *testing.T) {
	f := makeAttFix(t)
	admin := mustFindUser(t, f, "00000001")
	bearer := loginAsAdmin(t, f)

	status, env := doJSON(t, f.r, http.MethodDelete, "/api/v1/users/"+strconv.Itoa(int(admin.ID)), bearer, nil)
	if status != http.StatusUnprocessableEntity || !strings.Contains(env.Message, "akun sendiri") {
		t.Fatalf("delete self: %d %+v", status, env)
	}

	var count int64
	f.db.Unscoped().Model(&models.User{}).Where("id = ?", admin.ID).Count(&count)
	if count != 1 {
		t.Fatal("admin row must survive")
	}
}

func TestLastAdminProtectedFromDeleteAndDemote(t *testing.T) {
	f := makeAttFix(t)
	admin := mustFindUser(t, f, "00000001")
	bearer := loginAsAdmin(t, f)
	path := "/api/v1/users/" + strconv.Itoa(int(admin.ID))

	delStatus, delEnv := doJSON(t, f.r, http.MethodDelete, path, bearer, nil)
	if delStatus != http.StatusUnprocessableEntity || !strings.Contains(delEnv.Message, "akun sendiri") {
		t.Fatalf("delete self as sole admin: %d %+v", delStatus, delEnv)
	}

	demoteStatus, demoteEnv := doJSON(t, f.r, http.MethodPut, path, bearer, map[string]any{
		"name": admin.Name, "identity_number": admin.IdentityNumber,
		"email": "", "role": "mahasiswa",
	})
	if demoteStatus != http.StatusUnprocessableEntity || !strings.Contains(demoteEnv.Message, "Admin terakhir") {
		t.Fatalf("demote last admin: %d %+v", demoteStatus, demoteEnv)
	}

	createStatus, createEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/users", bearer, map[string]any{
		"name": "Admin Kedua", "identity_number": "00000002",
		"password": "rahasia123", "role": "admin",
	})
	if createStatus != http.StatusCreated {
		t.Fatalf("create second admin: %d %+v", createStatus, createEnv)
	}

	// Kini ada dua admin: yang pertama BOLEH menurunkan dirinya sendiri.
	demote2Status, _ := doJSON(t, f.r, http.MethodPut, path, bearer, map[string]any{
		"name": admin.Name, "identity_number": admin.IdentityNumber,
		"email": "", "role": "mahasiswa",
	})
	if demote2Status != http.StatusOK {
		t.Fatalf("demote self once another admin exists must succeed, got %d", demote2Status)
	}
}

func TestAttendanceRateLimitIsPerUserNotJustPerIP(t *testing.T) {
	f := makeAttFix(t)
	bearerA := bearerFor(t, f, f.student)
	bearerB := bearerFor(t, f, f.outsider)

	body := geoBody(f.active.ID, -6.5, 107.0) // jauh di luar radius → selalu ditolak, tak ganggu slot hadir

	send := func(bearer, ip string) int {
		code, _ := doJSONFrom(f.r, http.MethodPost, "/api/v1/attendance", bearer, body, ip+":1234")
		return code
	}

	for i := 0; i < 5; i++ {
		if code := send(bearerA, "10.9.0.1"); code == http.StatusTooManyRequests {
			t.Fatalf("request A #%d must not be limited yet", i+1)
		}
		if code := send(bearerA, "10.9.0.2"); code == http.StatusTooManyRequests {
			t.Fatalf("request A' #%d must not be limited yet", i+1)
		}
	}

	if code := send(bearerA, "10.9.0.3"); code != http.StatusTooManyRequests {
		t.Fatalf("user A request #11 from fresh IP must be 429 (per-user bucket), got %d", code)
	}

	if code := send(bearerB, "10.9.0.1"); code == http.StatusTooManyRequests {
		t.Fatal("user B sharing IP with A must have its own bucket")
	}
}

func TestLateSubmissionMarkedTerlambat(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	now := time.Now().In(f.tz)
	late := models.Schedule{
		ClassID: f.class.ID, CourseID: f.active.CourseID, LocationID: f.loc.ID,
		StartTime: now.Add(-65 * time.Minute), EndTime: now.Add(-2 * time.Minute), // dalam toleransi +5m tapi lewat end
	}
	if err := f.db.Create(&late).Error; err != nil {
		t.Fatalf("create late schedule: %v", err)
	}

	status, env, data := submitAttendance(t, f, bearer, geoBody(late.ID, -6.20005, 106.816666))
	if status != http.StatusCreated || data.Status != models.StatusTerlambat {
		t.Fatalf("late submit = %d %s; %+v", status, data.Status, env)
	}
	if !strings.Contains(data.Message, "terlambat") {
		t.Fatalf("message must mention terlambat: %q", data.Message)
	}

	retryStatus, retryEnv, _ := submitAttendance(t, f, bearer, geoBody(late.ID, -6.20005, 106.816666))
	if retryStatus != http.StatusUnprocessableEntity || !strings.Contains(retryEnv.Message, "sudah") {
		t.Fatalf("terlambat must lock the slot: %d %+v", retryStatus, retryEnv)
	}

	var count int64
	f.db.Model(&models.Attendance{}).Where("schedule_id = ? AND status = ?", late.ID, models.StatusTerlambat).Count(&count)
	if count != 1 {
		t.Fatalf("terlambat rows = %d, want 1", count)
	}
}

func TestGpsAccuracyTooLowRejected(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	body := geoBody(f.active.ID, -6.20005, 106.816666)
	body["accuracy"] = 800.0

	status, env, data := submitAttendance(t, f, bearer, body)
	if status != http.StatusOK || data.Status != models.StatusDitolak {
		t.Fatalf("bad accuracy = %d %s; %+v", status, data.Status, env)
	}
	if !strings.Contains(data.Message, "akurasi") {
		t.Fatalf("message must mention akurasi: %q", data.Message)
	}

	var row models.Attendance
	if err := f.db.Where("schedule_id = ? AND user_id = ?", f.active.ID, f.student.ID).
		Order("id DESC").First(&row).Error; err != nil {
		t.Fatalf("ditolak row missing: %v", err)
	}
	if row.GpsAccuracy == nil || *row.GpsAccuracy != 800 {
		t.Fatalf("gps_accuracy not stored: %v", row.GpsAccuracy)
	}

	cleanStatus, _, cleanData := submitAttendance(t, f, bearer, geoBody(f.active.ID, -6.20005, 106.816666))
	if cleanStatus != http.StatusCreated || cleanData.Status != models.StatusHadir {
		t.Fatalf("retry without accuracy must be hadir: %d %s", cleanStatus, cleanData.Status)
	}
}

func TestImpossibleTravelRejected(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	var otherLoc models.Location
	if err := f.db.Where("name = ?", "Ruang Lain").First(&otherLoc).Error; err != nil {
		t.Fatalf("fixture otherLoc missing: %v", err)
	}
	now := time.Now().In(f.tz)
	far := models.Schedule{
		ClassID: f.class.ID, CourseID: f.active.CourseID, LocationID: otherLoc.ID,
		StartTime: now.Add(-15 * time.Minute), EndTime: now.Add(45 * time.Minute),
	}
	if err := f.db.Create(&far).Error; err != nil {
		t.Fatalf("create far schedule: %v", err)
	}

	hadirStatus, _, hadirData := submitAttendance(t, f, bearer, geoBody(f.active.ID, -6.20005, 106.816666))
	if hadirStatus != http.StatusCreated || hadirData.Status != models.StatusHadir {
		t.Fatalf("first hadir failed: %d %s", hadirStatus, hadirData.Status)
	}

	// Detik kemudian klaim hadir di lokasi lain ±14 km → kecepatan mustahil.
	jumpStatus, jumpEnv, jumpData := submitAttendance(t, f, bearer, geoBody(far.ID, -6.3, 106.9))
	if jumpStatus != http.StatusOK || jumpData.Status != models.StatusDitolak {
		t.Fatalf("impossible travel = %d %s; %+v", jumpStatus, jumpData.Status, jumpEnv)
	}
	if !strings.Contains(jumpData.Message, "perpindahan") {
		t.Fatalf("message must mention perpindahan: %q", jumpData.Message)
	}

	var rejected models.Attendance
	if err := f.db.Where("schedule_id = ? AND status = ?", far.ID, models.StatusDitolak).
		First(&rejected).Error; err != nil {
		t.Fatalf("audit row for impossible travel missing: %v", err)
	}
}

func mkRecurring(f attFix, days []int, start, end time.Time) models.Schedule {
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = strconv.Itoa(d)
	}
	csv := strings.Join(parts, ",")
	sched := models.Schedule{
		ClassID: f.class.ID, CourseID: f.active.CourseID, LocationID: f.loc.ID,
		StartTime: start, EndTime: end, IsRecurring: true, RepeatDays: &csv,
	}
	return sched
}

func TestRecurringScheduleWindow(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	now := time.Now().In(f.tz)

	// Seri yang jatuh HARI INI dengan window mencakup sekarang.
	today := mkRecurring(f, []int{int(now.Weekday())}, now.Add(-30*time.Minute), now.Add(60*time.Minute))
	if err := f.db.Create(&today).Error; err != nil {
		t.Fatalf("create recurring today: %v", err)
	}

	status, env, data := submitAttendance(t, f, bearer, geoBody(today.ID, -6.20005, 106.816666))
	if status != http.StatusCreated || data.Status != models.StatusHadir {
		t.Fatalf("recurring today attendance = %d %s; %+v", status, data.Status, env)
	}

	// Seri yang TIDAK jatuh hari ini → window tidak aktif.
	otherDay := (int(now.Weekday()) + 1) % 7
	away := mkRecurring(f, []int{otherDay}, now.Add(-30*time.Minute), now.Add(60*time.Minute))
	if err := f.db.Create(&away).Error; err != nil {
		t.Fatalf("create recurring otherday: %v", err)
	}

	deniedStatus, deniedEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/attendance", bearer,
		geoBody(away.ID, -6.20005, 106.816666))
	if deniedStatus != http.StatusUnprocessableEntity || !strings.Contains(deniedEnv.Message, "waktu jadwal aktif") {
		t.Fatalf("wrong-day recurring must be inactive: %d %+v", deniedStatus, deniedEnv)
	}
}

func TestTodaySchedulesIncludesRecurringSeries(t *testing.T) {
	f := makeAttFix(t)
	bearer := bearerFor(t, f, f.student)

	now := time.Now().In(f.tz)
	// Truncate ke detik: MySQL membulatkan pecahan detik saat menyimpan.
	seriesStart := now.Add(-15 * time.Minute).Truncate(time.Second)
	seriesEnd := now.Add(45 * time.Minute).Truncate(time.Second)
	series := mkRecurring(f, []int{int(now.Weekday())}, seriesStart, seriesEnd)
	if err := f.db.Create(&series).Error; err != nil {
		t.Fatalf("create series: %v", err)
	}

	status, env := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/today", bearer, nil)
	if status != http.StatusOK {
		t.Fatalf("today status = %d; %+v", status, env)
	}
	var items []struct {
		ID        uint   `json:"id"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		IsActive  bool   `json:"is_active"`
	}
	json.Unmarshal(decodeData(t, env), &items)

	for _, it := range items {
		if it.ID != series.ID {
			continue
		}
		if it.StartTime != seriesStart.Format("2006-01-02 15:04:05") ||
			it.EndTime != seriesEnd.Format("2006-01-02 15:04:05") || !it.IsActive {
			t.Fatalf("recurring item wrong: %+v (want %s..%s)", it,
				seriesStart.Format("2006-01-02 15:04:05"), seriesEnd.Format("2006-01-02 15:04:05"))
		}
		return
	}
	t.Fatalf("recurring series %d missing from today list: %+v", series.ID, items)
}

func TestRecurringFieldsRoundTripViaAdminApi(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	now := time.Now().In(f.tz)
	createStatus, createEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/schedules", admin, map[string]any{
		"class_id": f.class.ID, "course_id": f.active.CourseID, "location_id": f.loc.ID,
		"start_time":   now.Format("2006-01-02") + " 08:00:00",
		"end_time":     now.Format("2006-01-02") + " 10:00:00",
		"is_recurring": true,
		"repeat_days":  []int{1, 3, 5},
	})
	if createStatus != http.StatusCreated {
		t.Fatalf("create recurring: %d %+v", createStatus, createEnv)
	}
	var created struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(decodeData(t, createEnv), &created)

	showStatus, showEnv := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/"+strconv.Itoa(int(created.ID)), admin, nil)
	var got struct {
		IsRecurring bool  `json:"is_recurring"`
		RepeatDays  []int `json:"repeat_days"`
	}
	json.Unmarshal(decodeData(t, showEnv), &got)
	if showStatus != http.StatusOK || !got.IsRecurring || len(got.RepeatDays) != 3 || got.RepeatDays[0] != 1 || got.RepeatDays[2] != 5 {
		t.Fatalf("show recurring wrong: %d %+v", showStatus, got)
	}

	// Ubah menjadi sekali-jalan → field recurrence dibersihkan.
	updateStatus, updateEnv := doJSON(t, f.r, http.MethodPut, "/api/v1/schedules/"+strconv.Itoa(int(created.ID)), admin,
		scheduleReq(f.class.ID, f.active.CourseID, f.loc.ID,
			now.AddDate(0, 0, 3).Format("2006-01-02")+" 09:00:00",
			now.AddDate(0, 0, 3).Format("2006-01-02")+" 11:00:00"))
	if updateStatus != http.StatusOK {
		t.Fatalf("update to one-off: %d %+v", updateStatus, updateEnv)
	}
	show2Status, show2Env := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/"+strconv.Itoa(int(created.ID)), admin, nil)
	var got2 struct {
		IsRecurring bool    `json:"is_recurring"`
		RepeatDays  []int   `json:"repeat_days"`
		RecEnd      *string `json:"recurrence_end"`
	}
	json.Unmarshal(decodeData(t, show2Env), &got2)
	if show2Status != http.StatusOK || got2.IsRecurring || len(got2.RepeatDays) != 0 || got2.RecEnd != nil {
		t.Fatalf("one-off conversion must clear recurrence: %+v", got2)
	}

	// is_recurring tanpa repeat_days ditolak.
	badStatus, badEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/schedules", admin, map[string]any{
		"class_id": f.class.ID, "course_id": f.active.CourseID, "location_id": f.loc.ID,
		"start_time": "2026-01-05 08:00:00", "end_time": "2026-01-05 10:00:00",
		"is_recurring": true,
	})
	if badStatus != http.StatusUnprocessableEntity || badEnv.Errors == nil {
		t.Fatalf("recurring without days must 422: %d %+v", badStatus, badEnv)
	}
}
