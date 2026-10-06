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
	"github.com/edoazn/absensi-go/internal/testsupport"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type rawResponse struct {
	status      int
	contentType string
	body        []byte
}

func (r *rawResponse) ContentType() string { return r.contentType }
func (r *rawResponse) Body() []byte        { return r.body }

func doJSONRaw(t *testing.T, r *gin.Engine, bearer, path string) (int, *rawResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+bearer)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Code, &rawResponse{status: rec.Code, contentType: rec.Header().Get("Content-Type"), body: rec.Body.Bytes()}
}

func loginAsAdmin(t *testing.T, f attFix) string {
	t.Helper()
	return bearerFor(t, f, mustFindUser(t, f, "00000001"))
}

func loginAsOutsider(t *testing.T, f attFix) string {
	t.Helper()
	return bearerFor(t, f, f.outsider)
}

func mustFindUser(t *testing.T, f attFix, identity string) models.User {
	t.Helper()
	var user models.User
	if err := f.db.Where("identity_number = ?", identity).First(&user).Error; err != nil {
		t.Fatalf("find user %s: %v", identity, err)
	}
	return user
}

func TestLocationCrudRoundTrip(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	createStatus, createEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/locations", admin, map[string]any{
		"name": "Auditorium", "latitude": -6.201, "longitude": 106.817, "radius": 200,
	})
	if createStatus != http.StatusCreated {
		t.Fatalf("create status = %d; body %+v", createStatus, createEnv)
	}
	rawCreated, _ := json.Marshal(createEnv.Data)
	var created struct {
		ID uint `json:"id"`
	}
	json.Unmarshal(rawCreated, &created)
	if created.ID == 0 {
		t.Fatal("created id missing")
	}

	indexStatus, indexEnv := doJSON(t, f.r, http.MethodGet, "/api/v1/locations", admin, nil)
	if indexStatus != http.StatusOK {
		t.Fatalf("index status = %d", indexStatus)
	}
	rawIndex, _ := json.Marshal(indexEnv.Data)
	if !strings.Contains(string(rawIndex), "Auditorium") || !strings.Contains(string(rawIndex), "Lab PPLS") {
		t.Fatalf("index must contain seeded and new location: %s", rawIndex)
	}

	updateStatus, updateEnv := doJSON(t, f.r, http.MethodPut, "/api/v1/locations/"+strconv.Itoa(int(created.ID)), admin, map[string]any{
		"name": "Auditorium Lama", "latitude": -6.2015, "longitude": 106.8171, "radius": 250,
	})
	if updateStatus != http.StatusOK {
		t.Fatalf("update status = %d; body %+v", updateStatus, updateEnv)
	}
	if !strings.Contains(string(mustJSON(t, updateEnv)), "Auditorium Lama") {
		t.Fatal("updated name not returned")
	}
}

func TestLocationValidationErrors(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	status, env := doJSON(t, f.r, http.MethodPost, "/api/v1/locations", admin, map[string]any{
		"name": "", "latitude": 999,
	})

	if status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", status)
	}
	if env.Errors == nil {
		t.Fatal("errors bag required")
	}
}

func TestAdminEndpointsForbiddenForMahasiswa(t *testing.T) {
	f := makeAttFix(t)
	mahasiswa := loginAsOutsider(t, f)

	endpoints := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/locations"},
		{http.MethodPost, "/api/v1/locations"},
		{http.MethodGet, "/api/v1/schedules"},
		{http.MethodPost, "/api/v1/schedules"},
		{http.MethodGet, "/api/v1/reports/attendance"},
	}
	for _, ep := range endpoints {
		status, env := doJSON(t, f.r, ep.method, ep.path, mahasiswa, nil)
		if status != http.StatusForbidden {
			t.Fatalf("%s %s for mahasiswa = %d, want 403; %+v", ep.method, ep.path, status, env)
		}
	}
}

func TestScheduleCreateRoundTripAndValidation(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	badFkStatus, badFkEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/schedules", admin, map[string]any{
		"class_id": 999, "course_id": 1, "location_id": f.loc.ID,
		"start_time": "2026-09-01 08:00:00", "end_time": "2026-09-01 10:00:00",
	})
	if badFkStatus != http.StatusUnprocessableEntity || badFkEnv.Message != "Kelas tidak ditemukan" {
		t.Fatalf("unknown class: %d %+v", badFkStatus, badFkEnv)
	}

	badRangeStatus, _ := doJSON(t, f.r, http.MethodPost, "/api/v1/schedules", admin, map[string]any{
		"class_id": f.class.ID, "course_id": f.active.CourseID, "location_id": f.loc.ID,
		"start_time": "2026-09-01 10:00:00", "end_time": "2026-09-01 08:00:00",
	})
	if badRangeStatus != http.StatusUnprocessableEntity {
		t.Fatalf("end before start must fail, got %d", badRangeStatus)
	}

	okStatus, okEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/schedules", admin, map[string]any{
		"class_id": f.class.ID, "course_id": f.active.CourseID, "location_id": f.loc.ID,
		"start_time": "2026-09-01 08:00", "end_time": "2026-09-01T10:00:00",
	})
	if okStatus != http.StatusCreated {
		t.Fatalf("valid create failed: %d %+v", okStatus, okEnv)
	}

	indexStatus, indexEnv := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules?class_id="+strconv.Itoa(int(f.class.ID)), admin, nil)
	rawIndex, _ := json.Marshal(indexEnv.Data)
	if indexStatus != http.StatusOK || !strings.Contains(string(rawIndex), "GO301") {
		t.Fatalf("index must show new schedule: %d %s", indexStatus, rawIndex)
	}
}

func TestGenerateCodeFlow(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	path := "/api/v1/schedules/" + strconv.Itoa(int(f.codeSched.ID)) + "/generate-code"
	status, env := doJSON(t, f.r, http.MethodPost, path, admin, map[string]any{"minutes_valid": 15})
	if status != http.StatusOK {
		t.Fatalf("generate-code status = %d; body %+v", status, env)
	}
	raw, _ := json.Marshal(env.Data)
	var data struct {
		Code         string `json:"code"`
		ExpiresAt    string `json:"expires_at"`
		MinutesValid int    `json:"minutes_valid"`
	}
	json.Unmarshal(raw, &data)

	if len(data.Code) != 6 || data.Code != strings.ToUpper(data.Code) {
		t.Fatalf("code invalid: %q", data.Code)
	}
	if data.MinutesValid != 15 {
		t.Fatalf("minutes_valid = %d, want 15", data.MinutesValid)
	}

	var updated models.Schedule
	f.db.First(&updated, f.codeSched.ID)
	if updated.AttendanceCode == nil || *updated.AttendanceCode != data.Code {
		t.Fatal("code not persisted")
	}

	defaultStatus, defaultEnv := doJSON(t, f.r, http.MethodPost, path, admin, nil)
	rawDef, _ := json.Marshal(defaultEnv.Data)
	var defData struct {
		MinutesValid int `json:"minutes_valid"`
	}
	json.Unmarshal(rawDef, &defData)
	if defaultStatus != http.StatusOK || defData.MinutesValid != 30 {
		t.Fatalf("default minutes_valid must be 30: %d %+v", defaultStatus, defaultEnv)
	}

	invalidStatus, _ := doJSON(t, f.r, http.MethodPost, path, admin, map[string]any{"minutes_valid": 5000})
	if invalidStatus != http.StatusUnprocessableEntity {
		t.Fatalf("minutes_valid > 1440 must be 422, got %d", invalidStatus)
	}
}

func TestGenerateQrRotatesToken(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)
	before := *f.qrSched.QrToken

	path := "/api/v1/schedules/" + strconv.Itoa(int(f.qrSched.ID)) + "/generate-qr"
	status, env := doJSON(t, f.r, http.MethodPost, path, admin, nil)
	if status != http.StatusOK {
		t.Fatalf("generate-qr status = %d; body %+v", status, env)
	}

	var updated models.Schedule
	f.db.First(&updated, f.qrSched.ID)
	if updated.QrToken == nil || *updated.QrToken == before {
		t.Fatalf("token must rotate: old=%s new=%v", before, updated.QrToken)
	}
	if !strings.Contains(string(mustJSON(t, env)), *updated.QrToken) {
		t.Fatal("response must contain new token")
	}
}

func TestReportFilteringAndExport(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	now := time.Now().In(f.tz)
	oldDay := now.AddDate(0, 0, -3).Format("2006-01-02")
	todayDay := now.Format("2006-01-02")
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, f.tz)

	oldRow := models.Attendance{
		UserID: f.student.ID, ScheduleID: f.past.ID,
		Status: models.StatusDitolak, Method: models.MethodGeolocation,
		Latitude: testsupport.PtrFloat(-6.21), Longitude: testsupport.PtrFloat(106.83),
		Distance: testsupport.PtrFloat(1112), CreatedAt: now.AddDate(0, 0, -3),
	}
	todayRow := models.Attendance{
		UserID: f.student.ID, ScheduleID: f.active.ID,
		Status: models.StatusHadir, Method: models.MethodQrCode,
		CreatedAt: dayStart.Add(12 * time.Hour),
	}
	f.db.Create(&oldRow)
	f.db.Create(&todayRow)

	allStatus, allEnv := doJSON(t, f.r, http.MethodGet, "/api/v1/reports/attendance", admin, nil)
	if allStatus != http.StatusOK {
		t.Fatalf("report status = %d; body %+v", allStatus, allEnv)
	}
	totalItems := lenReportItems(t, allEnv)
	if totalItems != 2 {
		t.Fatalf("unfiltered report must have 2 rows, got %d", totalItems)
	}

	schedFiltered := lenReportItems(t, mustDo(t, f, admin, "/api/v1/reports/attendance?schedule_id="+strconv.Itoa(int(f.past.ID))))
	if schedFiltered != 1 {
		t.Fatalf("schedule filter must yield 1 row, got %d", schedFiltered)
	}

	rangeFiltered := lenReportItems(t, mustDo(t, f, admin, "/api/v1/reports/attendance?start_date="+oldDay+"&end_date="+oldDay))
	if rangeFiltered != 1 {
		t.Fatalf("date filter must yield 1 row, got %d", rangeFiltered)
	}

	exportStatus, exportEnv := doJSONRaw(t, f.r, admin, "/api/v1/reports/attendance/export?start_date="+todayDay+"&end_date="+todayDay)
	if exportStatus != http.StatusOK {
		t.Fatalf("export status = %d", exportStatus)
	}
	if !strings.Contains(exportEnv.ContentType(), "spreadsheetml") {
		t.Fatalf("export content type wrong: %s", exportEnv.ContentType())
	}
	if len(exportEnv.Body()) < 100 || string(exportEnv.Body()[:2]) != "PK" {
		t.Fatalf("export body must be a real xlsx (zip), got %d bytes", len(exportEnv.Body()))
	}

	workbook, err := excelize.OpenReader(bytes.NewReader(exportEnv.Body()))
	if err != nil {
		t.Fatalf("open xlsx: %v", err)
	}
	headerB1, _ := workbook.GetCellValue("Absensi", "B1")
	if headerB1 != "Nama Mahasiswa" {
		t.Fatalf("xlsx B1 = %q, want Nama Mahasiswa", headerB1)
	}
	statusD2, _ := workbook.GetCellValue("Absensi", "H2")
	if statusD2 != "Hadir" {
		t.Fatalf("xlsx H2 = %q, want Hadir (today row only)", statusD2)
	}
}

func mustDo(t *testing.T, f attFix, bearer, path string) api.Envelope {
	t.Helper()
	status, env := doJSON(t, f.r, http.MethodGet, path, bearer, nil)
	if status != http.StatusOK {
		t.Fatalf("GET %s = %d: %+v", path, status, env)
	}
	return env
}

type reportPayload struct {
	Items []map[string]any `json:"items"`
	Meta  struct {
		CurrentPage int   `json:"current_page"`
		PerPage     int   `json:"per_page"`
		Total       int64 `json:"total"`
		LastPage    int   `json:"last_page"`
	} `json:"meta"`
}

func lenReportItems(t *testing.T, env api.Envelope) int {
	t.Helper()
	raw, _ := json.Marshal(env.Data)
	var p reportPayload
	json.Unmarshal(raw, &p)
	return int(p.Meta.Total)
}

func mustJSON(t *testing.T, env api.Envelope) []byte {
	t.Helper()
	raw, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func TestReportPagination(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	now := time.Now().In(f.tz)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, f.tz)
	schedules := []uint{f.active.ID, f.past.ID, f.qrSched.ID}
	for i, schedID := range schedules {
		row := models.Attendance{
			UserID: f.student.ID, ScheduleID: schedID,
			Status: models.StatusHadir, Method: models.MethodQrCode,
			CreatedAt: dayStart.Add(time.Duration(8+i) * time.Hour),
		}
		if err := f.db.Create(&row).Error; err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}

	env := mustDo(t, f, admin, "/api/v1/reports/attendance?per_page=2&page=1")
	raw, _ := json.Marshal(env.Data)
	var page1 reportPayload
	json.Unmarshal(raw, &page1)
	if len(page1.Items) != 2 || page1.Meta.Total != 3 || page1.Meta.LastPage != 2 {
		t.Fatalf("page1 wrong: %s", raw)
	}
	env2 := mustDo(t, f, admin, "/api/v1/reports/attendance?per_page=2&page=2")
	raw2, _ := json.Marshal(env2.Data)
	var page2 reportPayload
	json.Unmarshal(raw2, &page2)
	if len(page2.Items) != 1 || page2.Meta.CurrentPage != 2 {
		t.Fatalf("page2 wrong: %s", raw2)
	}
}
