package models

import (
	"time"
)

// APIRequest represents the api_requests table structure for rate limiting and analytics
type APIRequest struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       *string   `gorm:"type:uuid;index" json:"user_id,omitempty"`
	IPAddress    string    `gorm:"type:varchar(45);not null" json:"ip_address"`
	Method       string    `gorm:"type:varchar(10);not null" json:"method"`
	Endpoint     string    `gorm:"type:varchar(255);not null;index" json:"endpoint"`
	StatusCode   int       `gorm:"not null" json:"status_code"`
	ResponseTime int       `gorm:"not null" json:"response_time"` // milliseconds
	RequestSize  int       `gorm:"not null;default:0" json:"request_size"`
	ResponseSize int       `gorm:"not null;default:0" json:"response_size"`
	UserAgent    string    `gorm:"type:text" json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`

	// Foreign key relationship
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName specifies the table name for GORM
func (APIRequest) TableName() string {
	return "api_requests"
}

// APIRequestCreateRequest represents the request payload for creating an API request log
type APIRequestCreateRequest struct {
	UserID       *string `json:"user_id,omitempty"`
	IPAddress    string  `json:"ip_address" validate:"required"`
	Method       string  `json:"method" validate:"required"`
	Endpoint     string  `json:"endpoint" validate:"required"`
	StatusCode   int     `json:"status_code"`
	ResponseTime int     `json:"response_time"`
	RequestSize  int     `json:"request_size"`
	ResponseSize int     `json:"response_size"`
	UserAgent    string  `json:"user_agent"`
}

// APIRequestResponse represents the response payload for API request operations
type APIRequestResponse struct {
	ID           uint   `json:"id"`
	UserID       *string `json:"user_id,omitempty"` // UUID as string
	IPAddress    string  `json:"ip_address"`
	Method       string  `json:"method"`
	Endpoint     string  `json:"endpoint"`
	StatusCode   int     `json:"status_code"`
	ResponseTime int     `json:"response_time"`
	RequestSize  int     `json:"request_size"`
	ResponseSize int     `json:"response_size"`
	UserAgent    string  `json:"user_agent"`
	CreatedAt    string  `json:"created_at"` // Formatted as ISO 8601
}

// APIRequestListResponse represents the response for paginated API request list
type APIRequestListResponse struct {
	Requests []APIRequestResponse `json:"requests"`
	Total    int                  `json:"total"`
	Page     int                  `json:"page"`
	Limit    int                  `json:"limit"`
	Pages    int                  `json:"pages"`
}

// APIAnalyticsResponse represents analytics data for API usage
type APIAnalyticsResponse struct {
	TotalRequests     int                    `json:"total_requests"`
	UniqueIPs         int                    `json:"unique_ips"`
	AverageResponseMs float64                `json:"average_response_ms"`
	RateLimitedCount  int                    `json:"rate_limited_count"`
	TopEndpoints      []EndpointStats        `json:"top_endpoints"`
	StatusCodeStats   map[string]int         `json:"status_code_stats"`
	HourlyStats       []HourlyRequestStats   `json:"hourly_stats"`
	Period            string                 `json:"period"` // e.g., "last_24h", "last_7d"
}

// EndpointStats represents statistics for a specific endpoint
type EndpointStats struct {
	Endpoint      string  `json:"endpoint"`
	RequestCount  int     `json:"request_count"`
	AverageTimeMs float64 `json:"average_time_ms"`
	ErrorRate     float64 `json:"error_rate"` // Percentage of 4xx/5xx responses
}

// HourlyRequestStats represents request statistics per hour
type HourlyRequestStats struct {
	Hour         string `json:"hour"`         // ISO 8601 hour format
	RequestCount int    `json:"request_count"`
	ErrorCount   int    `json:"error_count"`
	AvgTimeMs    float64 `json:"avg_time_ms"`
}

// RateLimitStatus represents the current rate limit status for an IP or user
type RateLimitStatus struct {
	IPAddress      string    `json:"ip_address"`
	UserID         *string   `json:"user_id,omitempty"`
	RequestCount   int       `json:"request_count"`
	WindowStart    time.Time `json:"window_start"`
	WindowEnd      time.Time `json:"window_end"`
	Limit          int       `json:"limit"`
	Remaining      int       `json:"remaining"`
	ResetTime      time.Time `json:"reset_time"`
	IsRateLimited  bool      `json:"is_rate_limited"`
}

// BeforeCreate is a GORM hook that runs before creating a record
func (ar *APIRequest) BeforeCreate() error {
	// GORM will handle auto-increment for ID
	return nil
}

// ToResponse converts APIRequest model to APIRequestResponse
func (ar *APIRequest) ToResponse() APIRequestResponse {
	response := APIRequestResponse{
		ID:           ar.ID,
		UserID:       ar.UserID,
		IPAddress:    ar.IPAddress,
		Method:       ar.Method,
		Endpoint:     ar.Endpoint,
		StatusCode:   ar.StatusCode,
		ResponseTime: ar.ResponseTime,
		RequestSize:  ar.RequestSize,
		ResponseSize: ar.ResponseSize,
		UserAgent:    ar.UserAgent,
		CreatedAt:    ar.CreatedAt.Format(time.RFC3339),
	}

	return response
}

// IsError checks if the API request resulted in an error (4xx or 5xx status code)
func (ar *APIRequest) IsError() bool {
	return ar.StatusCode >= 400
}

// IsClientError checks if the API request resulted in a client error (4xx status code)
func (ar *APIRequest) IsClientError() bool {
	return ar.StatusCode >= 400 && ar.StatusCode < 500
}

// IsServerError checks if the API request resulted in a server error (5xx status code)
func (ar *APIRequest) IsServerError() bool {
	return ar.StatusCode >= 500
}

// IsSlow checks if the API request took longer than the specified threshold (in milliseconds)
func (ar *APIRequest) IsSlow(thresholdMs int) bool {
	return ar.ResponseTime > thresholdMs
}

// GetEndpointGroup returns a simplified endpoint group for analytics
// e.g., "/api/images/123" -> "/api/images/{id}"
func (ar *APIRequest) GetEndpointGroup() string {
	// This would contain logic to group similar endpoints
	// For now, return the endpoint as-is
	return ar.Endpoint
}