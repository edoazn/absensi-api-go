package dto

type LocationRequest struct {
	Name      string   `json:"name" binding:"required"`
	Latitude  *float64 `json:"latitude" binding:"required,gte=-90,lte=90"`
	Longitude *float64 `json:"longitude" binding:"required,gte=-180,lte=180"`
	Radius    *float64 `json:"radius" binding:"required,gt=0"`
}

type ScheduleRequest struct {
	ClassID       uint   `json:"class_id" binding:"required"`
	CourseID      uint   `json:"course_id" binding:"required"`
	LocationID    uint   `json:"location_id" binding:"required"`
	StartTime     string `json:"start_time" binding:"required"`
	EndTime       string `json:"end_time" binding:"required"`
	IsRecurring   bool   `json:"is_recurring"`
	RepeatDays    []int  `json:"repeat_days" binding:"omitempty,dive,gte=0,lte=6"`
	RecurrenceEnd string `json:"recurrence_end" binding:"omitempty,datetime=2006-01-02"`
}

type GenerateCodeRequest struct {
	MinutesValid *int `json:"minutes_valid" binding:"omitempty,min=1,max=1440"`
}

type UserRequest struct {
	Name           string `json:"name" binding:"required"`
	IdentityNumber string `json:"identity_number" binding:"required"`
	Email          string `json:"email" binding:"omitempty,email"`
	Password       string `json:"password"`
	Role           string `json:"role" binding:"required,oneof=admin mahasiswa"`
	ClassIDs       []uint `json:"class_ids"`
}

type ClassRequest struct {
	Name         string `json:"name" binding:"required"`
	AcademicYear string `json:"academic_year" binding:"required"`
	UserIDs      []uint `json:"user_ids"`
}

type CourseRequest struct {
	CourseName   string  `json:"course_name" binding:"required"`
	CourseCode   string  `json:"course_code" binding:"required"`
	LecturerName string  `json:"lecturer_name" binding:"required"`
	LocationRoom *string `json:"location_room"`
}
