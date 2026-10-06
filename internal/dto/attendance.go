package dto

type AttendanceRequest struct {
	ScheduleID     uint     `json:"schedule_id" binding:"required"`
	Method         string   `json:"method" binding:"required,oneof=geolocation qr_code attendance_code"`
	Latitude       *float64 `json:"latitude" binding:"omitempty,required_if=Method geolocation,required_if=Method qr_code"`
	Longitude      *float64 `json:"longitude" binding:"omitempty,required_if=Method geolocation,required_if=Method qr_code"`
	Accuracy       *float64 `json:"accuracy" binding:"omitempty,gte=0,lte=10000"`
	QrToken        string   `json:"qr_token" binding:"omitempty,required_if=Method qr_code"`
	AttendanceCode string   `json:"attendance_code" binding:"omitempty,required_if=Method attendance_code"`
}

type AttendanceItemResponse struct {
	ID          uint     `json:"id"`
	UserID      uint     `json:"user_id"`
	ScheduleID  uint     `json:"schedule_id"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	Distance    *float64 `json:"distance"`
	GpsAccuracy *float64 `json:"gps_accuracy"`
	Status      string   `json:"status"`
	Method      string   `json:"method"`
	CreatedAt   string   `json:"created_at"`
}

type AttendanceStoreData struct {
	Status     string             `json:"status"`
	Distance   *float64           `json:"distance"`
	Method     string             `json:"method"`
	Message    string             `json:"message"`
	Attendance AttendanceResponse `json:"attendance"`
}

type AttendanceResponse struct {
	ID          uint     `json:"id"`
	UserID      uint     `json:"user_id"`
	ScheduleID  uint     `json:"schedule_id"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	Distance    *float64 `json:"distance"`
	GpsAccuracy *float64 `json:"gps_accuracy"`
	Status      string   `json:"status"`
	Method      string   `json:"method"`
	CreatedAt   string   `json:"created_at"`
}
