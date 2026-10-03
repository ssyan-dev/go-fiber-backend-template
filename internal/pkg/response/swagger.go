package response

type SuccessResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"operation successful"`
}

type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Message string `json:"message" example:"error description"`
}

type TokenResponse struct {
	Success bool      `json:"success" example:"true"`
	Message string    `json:"message" example:"login successful"`
	Data    TokenData `json:"data"`
}

type TokenData struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
}

type UserResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"success"`
	Data    any    `json:"data"`
}

type HealthData struct {
	Status    string            `json:"status" example:"ok"`
	Env       string            `json:"env" example:"development"`
	Timestamp string            `json:"timestamp" example:"2026-10-03T18:00:00Z"`
	Services  map[string]string `json:"services"`
}

type HealthResponse struct {
	Success bool       `json:"success" example:"true"`
	Message string     `json:"message" example:"server is healthy"`
	Data    HealthData `json:"data"`
}
