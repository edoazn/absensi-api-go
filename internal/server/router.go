package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/edoazn/absensi-go/config"
	_ "github.com/edoazn/absensi-go/docs"
	"github.com/edoazn/absensi-go/internal/api"
	"github.com/edoazn/absensi-go/internal/handlers"
	"github.com/edoazn/absensi-go/internal/middleware"
	"github.com/edoazn/absensi-go/internal/services"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

type Deps struct {
	Cfg *config.Config
	DB  *gorm.DB
}

func BuildRouter(d Deps) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(middleware.CORS(d.Cfg.CorsOrigins))

	blacklist := services.NewBlacklist()
	tokens := services.NewTokenService(d.Cfg, d.DB)
	authHandler := handlers.NewAuthHandler(d.DB, tokens, blacklist)
	jwtAuth := middleware.JWTAuth(tokens, blacklist)

	attendanceService := services.NewAttendanceService(d.DB, d.Cfg.Timezone)
	scheduleService := services.NewScheduleService(d.DB, d.Cfg.Timezone)
	attendanceHandler := handlers.NewAttendanceHandler(d.DB, attendanceService, d.Cfg.Timezone)
	locationHandler := handlers.NewLocationHandler(d.DB)
	scheduleHandler := handlers.NewScheduleHandler(d.DB, scheduleService, d.Cfg.Timezone)
	reportHandler := handlers.NewReportHandler(d.DB, d.Cfg.Timezone)
	userHandler := handlers.NewUserHandler(d.DB, d.Cfg.BcryptCost)
	classHandler := handlers.NewClassHandler(d.DB)
	courseHandler := handlers.NewCourseHandler(d.DB)
	dashboardHandler := handlers.NewDashboardHandler(d.DB, d.Cfg.Timezone)

	attendanceRateLimit := middleware.NewRateLimit(6*time.Second, 10)
	attendanceUserLimit := middleware.NewRateLimit(6*time.Second, 10)
	loginRateLimit := middleware.NewRateLimit(6*time.Second, 10)

	userKey := func(c *gin.Context) string {
		return "u" + strconv.FormatUint(uint64(middleware.UserIDFrom(c)), 10)
	}

	// SPA admin dilayani di /admin — URL panel HTMX lama otomatis hidup lagi
	// sebagai rute asli React tanpa redirect.
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/admin/")
	})
	spa := spaHandler("./web/dist")
	router.GET("/admin", spa)
	router.GET("/admin/*filepath", spa)

	// Bookmark era prefix /static/admin dialihkan ke /admin.
	staticLegacy := redirectStaticAdminLegacy()
	router.Any("/static/admin", staticLegacy)
	router.Any("/static/admin/*rest", staticLegacy)

	// Rute tak dikenal: API tetap 404 JSON, sisanya 404 polos.
	router.NoRoute(noRouteFallback())

	router.GET("/healthz", healthz(d))

	// Swagger UI (spec dihasilkan `make swag` ke package docs).
	router.GET("/api/documentation", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/api/documentation/index.html")
	})
	router.GET("/api/documentation/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api/documentation/doc.json")))

	api := router.Group("/api/v1")
	api.GET("/", apiInfo(d))

	auth := api.Group("/auth")
	{
		auth.POST("/login", loginRateLimit.Middleware(), authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", jwtAuth, authHandler.Logout)
	}
	api.GET("/me", jwtAuth, authHandler.Me)

	protected := api.Group("", jwtAuth)
	{
		protected.POST("/attendance", attendanceRateLimit.Middleware(), attendanceUserLimit.MiddlewareForKey(userKey), attendanceHandler.Store)
		protected.GET("/attendance/history", attendanceHandler.History)
		protected.GET("/schedules/today", attendanceHandler.TodaySchedules)
	}

	admin := protected.Group("", middleware.RequireAdmin())
	{
		admin.GET("/admin/dashboard", dashboardHandler.Stats)

		admin.GET("/users", userHandler.Index)
		admin.POST("/users", userHandler.Store)
		admin.PUT("/users/:id", userHandler.Update)
		admin.DELETE("/users/:id", userHandler.Destroy)

		admin.GET("/classes", classHandler.Index)
		admin.POST("/classes", classHandler.Store)
		admin.PUT("/classes/:id", classHandler.Update)
		admin.DELETE("/classes/:id", classHandler.Destroy)

		admin.GET("/courses", courseHandler.Index)
		admin.POST("/courses", courseHandler.Store)
		admin.PUT("/courses/:id", courseHandler.Update)
		admin.DELETE("/courses/:id", courseHandler.Destroy)

		admin.GET("/locations", locationHandler.Index)
		admin.POST("/locations", locationHandler.Store)
		admin.PUT("/locations/:id", locationHandler.Update)

		admin.GET("/schedules", scheduleHandler.Index)
		admin.GET("/schedules/:id", scheduleHandler.Show)
		admin.POST("/schedules", scheduleHandler.Store)
		admin.PUT("/schedules/:id", scheduleHandler.Update)
		admin.DELETE("/schedules/:id", scheduleHandler.Destroy)
		admin.POST("/schedules/:id/generate-code", scheduleHandler.GenerateCode)
		admin.POST("/schedules/:id/generate-qr", scheduleHandler.GenerateQr)
		admin.GET("/schedules/:id/qr.png", scheduleHandler.QrPng)

		admin.GET("/reports/attendance", reportHandler.AttendanceReport)
		admin.GET("/reports/attendance/export", reportHandler.ExportExcel)
	}

	return router
}

func healthz(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, code, msg := "up", http.StatusOK, "ok"
		if sqlDB, err := d.DB.DB(); err != nil || sqlDB.Ping() != nil {
			status, code, msg = "down", http.StatusServiceUnavailable, "database unreachable"
		}
		c.JSON(code, gin.H{
			"success": status == "up",
			"data": gin.H{
				"name":     d.Cfg.AppName,
				"env":      d.Cfg.AppEnv,
				"database": status,
			},
			"message": msg,
		})
	}
}

// spaHandler melayani SPA React di bawah /admin: berkas yang ada dikirim
// langsung (aset /assets/ ber-hash → cache immutable), deep-link tanpa berkas
// difallback ke index.html agar React Router menangani rutenya.
// Path yang lolos keluar root dist ditolak sebagai mitigasi path traversal,
// dan path ber-ekstensi yang tak ditemukan 404 polos (bukan HTML disalahartikan
// sebagai JS/CSS).
func spaHandler(root string) gin.HandlerFunc {
	index := filepath.Join(root, "index.html")
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		rootAbs = root
	}
	return func(c *gin.Context) {
		path := c.Param("filepath")

		rel := filepath.FromSlash(strings.TrimPrefix(path, "/"))
		full, err := filepath.Abs(filepath.Join(root, rel))
		// Request "/admin" me-resolve tepat ke root itu sendiri (tanpa pemisah),
		// jadi kesetaraan juga dihitung — sisanya wajib berada DI DALAM root.
		withinRoot := err == nil &&
			strings.HasPrefix(full, rootAbs+string(os.PathSeparator))
		isRootItself := err == nil && strings.EqualFold(full, rootAbs)

		info, statErr := os.Stat(full)
		switch {
		case statErr == nil && !info.IsDir() && withinRoot:
			if strings.Contains(path, "/assets/") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				c.Header("Cache-Control", "no-cache")
			}
			c.File(full)
		case (!withinRoot && !isRootItself) || filepath.Ext(path) != "":
			c.Status(http.StatusNotFound)
		default:
			c.Header("Cache-Control", "no-cache")
			c.File(index)
		}
	}
}

// noRouteFallback menjaga kontrak API: permintaan API tak dikenal tetap
// 404 JSON envelope; sisanya 404 polos.
func noRouteFallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/api" || strings.HasPrefix(path, "/api/") {
			api.Fail(c, http.StatusNotFound, "Endpoint tidak ditemukan")
			return
		}
		c.Status(http.StatusNotFound)
	}
}

// redirectStaticAdminLegacy memetakan bookmark era prefix /static/admin/*
// ke /admin/* agar tab lama tidak mati. Sub-path tanpa padanan SPA
// diarahkan ke section root-nya; sisanya jatuh ke dashboard (/admin/).
func redirectStaticAdminLegacy() gin.HandlerFunc {
	sections := map[string]bool{
		"users":       true,
		"classes":     true,
		"courses":     true,
		"locations":   true,
		"schedules":   true,
		"attendances": true,
	}
	return func(c *gin.Context) {
		target := "/admin/"
		if rest := strings.Trim(c.Param("rest"), "/"); rest != "" {
			switch section, _, _ := strings.Cut(rest, "/"); {
			case rest == "login":
				target = "/admin/login"
			case sections[section]:
				target = "/admin/" + section
			}
		}
		c.Redirect(http.StatusFound, target)
	}
}

func apiInfo(d Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"name":          d.Cfg.AppName,
				"version":       "v1",
				"documentation": "/api/documentation",
			},
			"message": "Welcome to Absensi Mahasiswa API",
		})
	}
}
