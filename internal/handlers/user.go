package handlers

import (
	"net/http"
	"strings"

	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/dto"
	"github.com/edoazn/absensi-go/internal/middleware"
	"github.com/edoazn/absensi-go/internal/models"
	"github.com/edoazn/absensi-go/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	db         *gorm.DB
	bcryptCost int
}

func NewUserHandler(db *gorm.DB, bcryptCost int) *UserHandler {
	return &UserHandler{db: db, bcryptCost: bcryptCost}
}

// Index Daftar pengguna
// @Summary Daftar pengguna
// @Tags Admin: Users
// @Accept json
// @Produce json
// @Param role query string false "Filter role (admin|mahasiswa)"
// @Success 200 {object} api.Envelope
// @Security BearerAuth
// @Router /users [get]
func (h *UserHandler) Index(c *gin.Context) {
	var users []models.User
	query := h.db.Preload("Classes").Order("name ASC")
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if err := query.Find(&users).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memuat pengguna")
		return
	}
	api.OK(c, users, "Daftar pengguna")
}

func (h *UserHandler) validateUniqueness(req dto.UserRequest, excludeID uint) string {
	dupIdentity := h.db.Model(&models.User{}).Where("identity_number = ?", req.IdentityNumber)
	if excludeID > 0 {
		dupIdentity = dupIdentity.Where("id <> ?", excludeID)
	}
	var count int64
	dupIdentity.Count(&count)
	if count > 0 {
		return "NIM/NIP sudah terdaftar untuk pengguna lain."
	}
	if strings.TrimSpace(req.Email) != "" {
		dupEmail := h.db.Model(&models.User{}).Where("email = ?", strings.TrimSpace(req.Email))
		if excludeID > 0 {
			dupEmail = dupEmail.Where("id <> ?", excludeID)
		}
		dupEmail.Count(&count)
		if count > 0 {
			return "Email sudah digunakan pengguna lain."
		}
	}
	if excludeID == 0 && len(strings.TrimSpace(req.Password)) < 6 {
		return "Password minimal 6 karakter."
	}
	if excludeID > 0 && req.Password != "" && len(req.Password) < 6 {
		return "Password baru minimal 6 karakter."
	}
	return ""
}

func (h *UserHandler) syncClasses(tx *gorm.DB, userID uint, classIDs []uint) error {
	var user models.User
	if err := tx.First(&user, userID).Error; err != nil {
		return err
	}
	var classes []models.Class
	if len(classIDs) > 0 {
		if err := tx.Find(&classes, classIDs).Error; err != nil {
			return err
		}
	}
	return tx.Model(&user).Association("Classes").Replace(classes)
}

// Store Buat pengguna
// @Summary Buat pengguna
// @Tags Admin: Users
// @Accept json
// @Produce json
// @Param body body dto.UserRequest true "Pengguna"
// @Success 201 {object} api.Envelope
// @Failure 422 {object} api.Envelope
// @Security BearerAuth
// @Router /users [post]
func (h *UserHandler) Store(c *gin.Context) {
	var req dto.UserRequest
	if !api.BindJSON(c, &req) {
		return
	}
	if msg := h.validateUniqueness(req, 0); msg != "" {
		api.Fail(c, http.StatusUnprocessableEntity, msg)
		return
	}

	hashed, err := services.HashPassword(req.Password, h.bcryptCost)
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal memproses password")
		return
	}

	user := models.User{
		Name:           strings.TrimSpace(req.Name),
		IdentityNumber: strings.TrimSpace(req.IdentityNumber),
		Password:       hashed,
		Role:           req.Role,
	}
	if email := strings.TrimSpace(req.Email); email != "" {
		user.Email = &email
	}

	err = h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if user.IsMahasiswa() {
			return h.syncClasses(tx, user.ID, req.ClassIDs)
		}
		return nil
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menyimpan pengguna")
		return
	}
	h.db.Preload("Classes").First(&user, user.ID)
	recordAudit(h.db, c, "user.create", "user", &user.ID, map[string]any{
		"identity_number": user.IdentityNumber, "role": user.Role,
	})
	api.Created(c, user, "Pengguna berhasil ditambahkan")
}

// Update Ubah pengguna
// @Summary Ubah pengguna
// @Tags Admin: Users
// @Accept json
// @Produce json
// @Param id path int true "ID user"
// @Param body body dto.UserRequest true "Pengguna"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Failure 422 {object} api.Envelope
// @Security BearerAuth
// @Router /users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		api.Fail(c, http.StatusBadRequest, "ID pengguna tidak valid")
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}

	var req dto.UserRequest
	if !api.BindJSON(c, &req) {
		return
	}
	if msg := h.validateUniqueness(req, id); msg != "" {
		api.Fail(c, http.StatusUnprocessableEntity, msg)
		return
	}
	if !h.guardLastAdmin(c, id, user.Role, req.Role) {
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		user.Name = strings.TrimSpace(req.Name)
		user.IdentityNumber = strings.TrimSpace(req.IdentityNumber)
		email := strings.TrimSpace(req.Email)
		if email == "" {
			user.Email = nil
		} else {
			user.Email = &email
		}
		user.Role = req.Role
		if req.Password != "" {
			hashed, hashErr := services.HashPassword(req.Password, h.bcryptCost)
			if hashErr != nil {
				return hashErr
			}
			user.Password = hashed
		}
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		if user.IsMahasiswa() {
			return h.syncClasses(tx, user.ID, req.ClassIDs)
		}
		return h.syncClasses(tx, user.ID, nil)
	})
	if err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menyimpan perubahan pengguna")
		return
	}
	h.db.Preload("Classes").First(&user, id)
	recordAudit(h.db, c, "user.update", "user", &user.ID, map[string]any{
		"identity_number": user.IdentityNumber, "role": req.Role,
	})
	api.OK(c, user, "Pengguna berhasil diperbarui")
}

// guardLastAdmin mencegah admin terakhir kehilangan akses: baik karena
// dihapus maupun karena diturunkan perannya menjadi mahasiswa.
func (h *UserHandler) guardLastAdmin(c *gin.Context, excludeID uint, targetRole string, newRole string) bool {
	if targetRole != models.RoleAdmin || newRole == models.RoleAdmin {
		return true
	}
	var otherAdmins int64
	h.db.Model(&models.User{}).Where("role = ? AND id <> ?", models.RoleAdmin, excludeID).Count(&otherAdmins)
	if otherAdmins > 0 {
		return true
	}
	api.Fail(c, http.StatusUnprocessableEntity, "Admin terakhir tidak boleh dihapus atau diturunkan perannya")
	return false
}

// Destroy Hapus pengguna
// @Summary Hapus pengguna
// @Tags Admin: Users
// @Accept json
// @Produce json
// @Param id path int true "ID user"
// @Success 200 {object} api.Envelope
// @Failure 404 {object} api.Envelope
// @Security BearerAuth
// @Router /users/{id} [delete]
func (h *UserHandler) Destroy(c *gin.Context) {
	id, ok := paramUint(c, "id")
	if !ok {
		api.Fail(c, http.StatusBadRequest, "ID pengguna tidak valid")
		return
	}
	if caller := middleware.UserIDFrom(c); caller == id {
		api.Fail(c, http.StatusUnprocessableEntity, "Anda tidak dapat menghapus akun sendiri")
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		api.Fail(c, http.StatusNotFound, "Pengguna tidak ditemukan")
		return
	}
	if !h.guardLastAdmin(c, id, user.Role, "") {
		return
	}
	if err := h.db.Delete(&models.User{}, id).Error; err != nil {
		api.Fail(c, http.StatusInternalServerError, "Gagal menghapus pengguna")
		return
	}
	recordAudit(h.db, c, "user.delete", "user", &id, map[string]any{
		"identity_number": user.IdentityNumber,
	})
	api.OK(c, nil, "Pengguna berhasil dihapus")
}
