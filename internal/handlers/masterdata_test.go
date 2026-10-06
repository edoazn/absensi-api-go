package handlers_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/models"
)

type userPayload struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	IdentityNumber string  `json:"identity_number"`
	Email          *string `json:"email"`
	Role           string  `json:"role"`
	Classes        []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"classes"`
}

func decodeData(t *testing.T, env api.Envelope) []byte {
	t.Helper()
	raw, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	return raw
}

func TestUserCrudRoundTrip(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	createStatus, createEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"name": "Budi Mahasiswa", "identity_number": "301290001", "email": "budi@kampus.ac.id",
		"password": "rahasia123", "role": "mahasiswa", "class_ids": []uint{f.class.ID},
	})
	if createStatus != http.StatusCreated {
		t.Fatalf("create status = %d; body %+v", createStatus, createEnv)
	}
	var created userPayload
	json.Unmarshal(decodeData(t, createEnv), &created)
	if created.ID == 0 || created.Role != "mahasiswa" || len(created.Classes) != 1 || created.Classes[0].ID != f.class.ID {
		t.Fatalf("created payload wrong: %+v", created)
	}

	indexStatus, indexEnv := doJSON(t, f.r, http.MethodGet, "/api/v1/users?role=admin", admin, nil)
	if indexStatus != http.StatusOK {
		t.Fatalf("index status = %d", indexStatus)
	}
	var users []userPayload
	json.Unmarshal(decodeData(t, indexEnv), &users)
	for _, u := range users {
		if u.Role != "admin" {
			t.Fatalf("role filter leaked: %+v", u)
		}
	}

	updateStatus, updateEnv := doJSON(t, f.r, http.MethodPut, "/api/v1/users/"+strconv.Itoa(int(created.ID)), admin, map[string]any{
		"name": "Budi Santoso", "identity_number": "301290001", "email": "",
		"role": "mahasiswa", "class_ids": []uint{},
	})
	if updateStatus != http.StatusOK {
		t.Fatalf("update status = %d; body %+v", updateStatus, updateEnv)
	}
	var updated userPayload
	json.Unmarshal(decodeData(t, updateEnv), &updated)
	if updated.Name != "Budi Santoso" || updated.Email != nil || len(updated.Classes) != 0 {
		t.Fatalf("update payload wrong: %+v", updated)
	}

	deleteStatus, deleteEnv := doJSON(t, f.r, http.MethodDelete, "/api/v1/users/"+strconv.Itoa(int(created.ID)), admin, nil)
	if deleteStatus != http.StatusOK {
		t.Fatalf("delete status = %d; body %+v", deleteStatus, deleteEnv)
	}
	var count int64
	f.db.Unscoped().Model(&models.User{}).Where("id = ?", created.ID).Count(&count)
	if count != 1 {
		t.Fatal("expected soft-deleted row to remain")
	}
	f.db.Model(&models.User{}).Where("id = ?", created.ID).Count(&count)
	if count != 0 {
		t.Fatal("user not soft deleted")
	}
}

func TestUserValidationAndDuplicates(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	dupNimStatus, dupNimEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"name": "Dup NIM", "identity_number": "301220001", "password": "rahasia123", "role": "mahasiswa",
	})
	if dupNimStatus != http.StatusUnprocessableEntity || !strings.Contains(dupNimEnv.Message, "NIM/NIP") {
		t.Fatalf("dup nim: %d %+v", dupNimStatus, dupNimEnv)
	}

	badRoleStatus, badRoleEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"name": "Bad Role", "identity_number": "301290999", "password": "rahasia123", "role": "dosen",
	})
	if badRoleStatus != http.StatusUnprocessableEntity || badRoleEnv.Errors == nil {
		t.Fatalf("bad role: %d %+v", badRoleStatus, badRoleEnv)
	}

	shortPassStatus, shortPassEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"name": "Short Pass", "identity_number": "301290998", "password": "123", "role": "mahasiswa",
	})
	if shortPassStatus != http.StatusUnprocessableEntity || !strings.Contains(shortPassEnv.Message, "Password") {
		t.Fatalf("short pass: %d %+v", shortPassStatus, shortPassEnv)
	}

	createStatus, createEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/users", admin, map[string]any{
		"name": "Ada Email", "identity_number": "301290990", "email": "unique@kampus.ac.id",
		"password": "rahasia123", "role": "mahasiswa",
	})
	if createStatus != http.StatusCreated {
		t.Fatalf("create: %d %+v", createStatus, createEnv)
	}
	var made userPayload
	json.Unmarshal(decodeData(t, createEnv), &made)

	updateDupEmailStatus, updateDupEmailEnv := doJSON(t, f.r, http.MethodPut, "/api/v1/users/"+strconv.Itoa(int(made.ID)), admin, map[string]any{
		"name": "Ada Email", "identity_number": "301290990", "email": "budi2@kampus.ac.id", "role": "mahasiswa",
	})
	if updateDupEmailStatus == http.StatusUnprocessableEntity {
		t.Fatalf("unexpected dup on unique email: %+v", updateDupEmailEnv)
	}
}

func TestClassCrudRoundTrip(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	createStatus, createEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/classes", admin, map[string]any{
		"name": "TI-5Z", "academic_year": "2026/2027", "user_ids": []uint{f.student.ID},
	})
	if createStatus != http.StatusCreated {
		t.Fatalf("create status = %d; body %+v", createStatus, createEnv)
	}
	var created struct {
		ID       uint `json:"id"`
		Students []struct {
			ID uint `json:"id"`
		} `json:"students"`
		StudentIDs []uint `json:"-"`
	}
	json.Unmarshal(decodeData(t, createEnv), &created)
	if created.ID == 0 || len(created.Students) != 1 || created.Students[0].ID != f.student.ID {
		t.Fatalf("class students wrong: %+v", created)
	}

	updateStatus, updateEnv := doJSON(t, f.r, http.MethodPut, "/api/v1/classes/"+strconv.Itoa(int(created.ID)), admin, map[string]any{
		"name": "TI-5Y", "academic_year": "2026/2027", "user_ids": []uint{},
	})
	if updateStatus != http.StatusOK {
		t.Fatalf("update status = %d; body %+v", updateStatus, updateEnv)
	}
	var updated struct {
		Name     string `json:"name"`
		Students []struct {
			ID uint `json:"id"`
		} `json:"students"`
	}
	json.Unmarshal(decodeData(t, updateEnv), &updated)
	if updated.Name != "TI-5Y" || len(updated.Students) != 0 {
		t.Fatalf("updated class wrong: %+v", updated)
	}

	deleteStatus, _ := doJSON(t, f.r, http.MethodDelete, "/api/v1/classes/"+strconv.Itoa(int(created.ID)), admin, nil)
	if deleteStatus != http.StatusOK {
		t.Fatalf("delete status = %d", deleteStatus)
	}
}

func TestCourseCrudRoundTrip(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	createStatus, createEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/courses", admin, map[string]any{
		"course_name": "Basis Data", "course_code": "bd401", "lecturer_name": "Bu Data",
	})
	if createStatus != http.StatusCreated {
		t.Fatalf("create status = %d; body %+v", createStatus, createEnv)
	}
	var created models.Course
	json.Unmarshal(decodeData(t, createEnv), &created)
	if created.CourseCode != "BD401" {
		t.Fatalf("course code must be uppercased: %q", created.CourseCode)
	}

	dupStatus, dupEnv := doJSON(t, f.r, http.MethodPost, "/api/v1/courses", admin, map[string]any{
		"course_name": "Dup", "course_code": "BD401", "lecturer_name": "X",
	})
	if dupStatus != http.StatusUnprocessableEntity || !strings.Contains(dupEnv.Message, "sudah dipakai") {
		t.Fatalf("dup code: %d %+v", dupStatus, dupEnv)
	}

	updateStatus, updateEnv := doJSON(t, f.r, http.MethodPut, "/api/v1/courses/"+strconv.Itoa(int(created.ID)), admin, map[string]any{
		"course_name": "Basis Data Lanjut", "course_code": "BD401", "lecturer_name": "Bu Data", "location_room": "R-101",
	})
	if updateStatus != http.StatusOK {
		t.Fatalf("update status = %d; body %+v", updateStatus, updateEnv)
	}
	var updated models.Course
	json.Unmarshal(decodeData(t, updateEnv), &updated)
	if updated.CourseName != "Basis Data Lanjut" || updated.LocationRoom == nil || *updated.LocationRoom != "R-101" {
		t.Fatalf("updated course wrong: %+v", updated)
	}

	deleteStatus, _ := doJSON(t, f.r, http.MethodDelete, "/api/v1/courses/"+strconv.Itoa(int(created.ID)), admin, nil)
	if deleteStatus != http.StatusOK {
		t.Fatalf("delete status = %d", deleteStatus)
	}
}

func TestDashboardStatsEndpoint(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	status, env := doJSON(t, f.r, http.MethodGet, "/api/v1/admin/dashboard", admin, nil)
	if status != http.StatusOK {
		t.Fatalf("dashboard status = %d; body %+v", status, env)
	}
	raw := string(decodeData(t, env))
	for _, key := range []string{"totals", "recent", "schedules", `"users"`, `"courses"`, `"locations"`} {
		if !strings.Contains(raw, key) {
			t.Fatalf("dashboard missing key %s in %s", key, raw)
		}
	}
}

func TestQrPngViaJWT(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	pngStatus, pngResp := doJSONRaw(t, f.r, admin, "/api/v1/schedules/"+strconv.Itoa(int(f.qrSched.ID))+"/qr.png")
	if pngStatus != http.StatusOK || pngResp.ContentType() != "image/png" {
		t.Fatalf("qr png status/type wrong: %d %s", pngStatus, pngResp.ContentType())
	}
	body := pngResp.Body()
	if len(body) < 8 || body[0] != 0x89 || body[1] != 'P' || body[2] != 'N' || body[3] != 'G' {
		t.Fatal("not a PNG payload")
	}
}

type schedulePayload struct {
	ID         uint   `json:"id"`
	ClassID    uint   `json:"class_id"`
	CourseID   uint   `json:"course_id"`
	LocationID uint   `json:"location_id"`
	Class      string `json:"class"`
	Course     string `json:"course"`
	CourseCode string `json:"course_code"`
	Location   string `json:"location"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	IsActive   bool   `json:"is_active"`
	HasQr      bool   `json:"has_qr"`
}

func scheduleReq(classID, courseID, locationID uint, start, end string) map[string]any {
	return map[string]any{
		"class_id": classID, "course_id": courseID, "location_id": locationID,
		"start_time": start, "end_time": end,
	}
}

func TestScheduleShowEndpoint(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	status, env := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/"+strconv.Itoa(int(f.active.ID)), admin, nil)
	if status != http.StatusOK {
		t.Fatalf("show status = %d; body %+v", status, env)
	}
	var got schedulePayload
	json.Unmarshal(decodeData(t, env), &got)
	if got.ID != f.active.ID || got.ClassID != f.class.ID || got.Class != "TI-3A" ||
		got.CourseCode != "GO301" || got.Location != "Lab PPLS" || !got.IsActive {
		t.Fatalf("show payload wrong: %+v", got)
	}

	notFoundStatus, notFoundEnv := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/999999", admin, nil)
	if notFoundStatus != http.StatusNotFound || notFoundEnv.Message == "" {
		t.Fatalf("missing schedule: %d %+v", notFoundStatus, notFoundEnv)
	}

	badIDStatus, _ := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/abc", admin, nil)
	if badIDStatus != http.StatusBadRequest {
		t.Fatalf("invalid id status = %d, want 400", badIDStatus)
	}
}

func TestScheduleUpdateRoundTripAndValidation(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	var otherLoc models.Location
	if err := f.db.Where("name = ?", "Ruang Lain").First(&otherLoc).Error; err != nil {
		t.Fatalf("fixture otherLoc missing: %v", err)
	}

	newStart := time.Now().In(f.tz).AddDate(0, 0, 2).Format("2006-01-02") + " 08:00:00"
	newEnd := time.Now().In(f.tz).AddDate(0, 0, 2).Format("2006-01-02") + " 10:00:00"

	updateStatus, updateEnv := doJSON(t, f.r, http.MethodPut, "/api/v1/schedules/"+strconv.Itoa(int(f.active.ID)), admin,
		scheduleReq(f.class.ID, f.active.CourseID, otherLoc.ID, newStart, newEnd))
	if updateStatus != http.StatusOK {
		t.Fatalf("update status = %d; body %+v", updateStatus, updateEnv)
	}

	showStatus, showEnv := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/"+strconv.Itoa(int(f.active.ID)), admin, nil)
	if showStatus != http.StatusOK {
		t.Fatalf("show after update status = %d", showStatus)
	}
	var got schedulePayload
	json.Unmarshal(decodeData(t, showEnv), &got)
	if got.LocationID != otherLoc.ID || got.Location != "Ruang Lain" ||
		got.StartTime != newStart || got.EndTime != newEnd || got.IsActive {
		t.Fatalf("update not persisted: %+v", got)
	}

	badFKStatus, badFKEnv := doJSON(t, f.r, http.MethodPut, "/api/v1/schedules/"+strconv.Itoa(int(f.active.ID)), admin,
		scheduleReq(999999, f.active.CourseID, f.loc.ID, newStart, newEnd))
	if badFKStatus != http.StatusUnprocessableEntity || !strings.Contains(badFKEnv.Message, "Kelas") {
		t.Fatalf("bad fk: %d %+v", badFKStatus, badFKEnv)
	}

	reversedStatus, reversedEnv := doJSON(t, f.r, http.MethodPut, "/api/v1/schedules/"+strconv.Itoa(int(f.active.ID)), admin,
		scheduleReq(f.class.ID, f.active.CourseID, f.loc.ID, newEnd, newStart))
	if reversedStatus != http.StatusUnprocessableEntity || reversedEnv.Errors == nil {
		t.Fatalf("end<=start must 422 with errors: %d %+v", reversedStatus, reversedEnv)
	}

	badFmtStatus, _ := doJSON(t, f.r, http.MethodPut, "/api/v1/schedules/"+strconv.Itoa(int(f.active.ID)), admin,
		scheduleReq(f.class.ID, f.active.CourseID, f.loc.ID, "bukan-waktu", newEnd))
	if badFmtStatus != http.StatusUnprocessableEntity {
		t.Fatalf("bad time format status = %d, want 422", badFmtStatus)
	}
}

func TestScheduleDeleteSoftDeletesRow(t *testing.T) {
	f := makeAttFix(t)
	admin := loginAsAdmin(t, f)

	deleteStatus, deleteEnv := doJSON(t, f.r, http.MethodDelete, "/api/v1/schedules/"+strconv.Itoa(int(f.past.ID)), admin, nil)
	if deleteStatus != http.StatusOK {
		t.Fatalf("delete status = %d; body %+v", deleteStatus, deleteEnv)
	}

	var count int64
	f.db.Unscoped().Model(&models.Schedule{}).Where("id = ?", f.past.ID).Count(&count)
	if count != 1 {
		t.Fatal("expected soft-deleted row to remain")
	}
	f.db.Model(&models.Schedule{}).Where("id = ?", f.past.ID).Count(&count)
	if count != 0 {
		t.Fatal("schedule not soft deleted")
	}

	showAfterStatus, _ := doJSON(t, f.r, http.MethodGet, "/api/v1/schedules/"+strconv.Itoa(int(f.past.ID)), admin, nil)
	if showAfterStatus != http.StatusNotFound {
		t.Fatalf("show after delete status = %d, want 404", showAfterStatus)
	}

	deleteAgainStatus, _ := doJSON(t, f.r, http.MethodDelete, "/api/v1/schedules/"+strconv.Itoa(int(f.past.ID)), admin, nil)
	if deleteAgainStatus != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want 404", deleteAgainStatus)
	}
}
