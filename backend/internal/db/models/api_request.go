package models

import (
	"time"

	"github.com/google/uuid"
)

// APIRequest represents the api_requests table structure for rate limiting and analytics
type APIRequest struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID         *uuid.UUID `gorm:"type:uuid;foreignKey:UserID;references:ID" json:"user_id,omitempty"`
	IPAddress      string     `gorm:"type:varchar(45)" json:"ip_address"`
	Endpoint       string     `gorm:"type:varchar(200)" json:"endpoint"`
	Method         string     `gorm:"type:varchar(10)" json:"method"`
	StatusCode     int        `gorm:"type:integer" json:"status_code"`
	ResponseTimeMs int        `gorm:"type:integer" json:"response_time_ms"`
	UserAgent      *string    `gorm:"type:text" json:"user_agent,omitempty"`
	Parameters     *string    `gorm:"type:text" json:"parameters,omitempty"` // JSON string
	Timestamp      time.Time  `gorm:"type:timestamp with time zone;default:now()" json:"timestamp"`
	RateLimited    bool       `gorm:"type:boolean;default:false" json:"rate_limited"`

	// Foreign key relationship
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName specifies the table name for GORM
func (APIRequest) TableName() string {
	return "api_requests"
}

// APIRequestCreateRequest represents the request payload for creating an API request log
type APIRequestCreateRequest struct {
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	IPAddress      string     `json:"ip_address" validate:"required"`
	Endpoint       string     `json:"endpoint" validate:"required"`
	Method         string     `json:"method" validate:"required"`
	StatusCode     int        `json:"status_code"`
	ResponseTimeMs int        `json:"response_time_ms"`
	UserAgent      *string    `json:"user_agent,omitempty"`
	Parameters     *string    `json:"parameters,omitempty"`
	RateLimited    bool       `json:"rate_limited"`
}

// APIRequestResponse represents the response payload for API request operations
type APIRequestResponse struct {
	ID             uuid.UUID `json:"id"`
	UserID         *string   `json:"user_id,omitempty"` // UUID as string
	IPAddress      string    `json:"ip_address"`
	Endpoint       string    `json:"endpoint"`
	Method         string    `json:"method"`
	StatusCode     int       `json:"status_code"`
	ResponseTimeMs int       `json:"response_time_ms"`
	UserAgent      *string   `json:"user_agent,omitempty"`
	Parameters     *string   `json:"parameters,omitempty"`
	Timestamp      string    `json:"timestamp"` // Formatted as ISO 8601
	RateLimited    bool      `json:"rate_limited"`
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
	if ar.ID == uuid.Nil {
		ar.ID = uuid.New()
	}
	return nil
}

// ToResponse converts APIRequest model to APIRequestResponse
func (ar *APIRequest) ToResponse() APIRequestResponse {
	response := APIRequestResponse{
		ID:             ar.ID,
		IPAddress:      ar.IPAddress,
		Endpoint:       ar.Endpoint,
		Method:         ar.Method,
		StatusCode:     ar.StatusCode,
		ResponseTimeMs: ar.ResponseTimeMs,
		UserAgent:      ar.UserAgent,
		Parameters:     ar.Parameters,
		Timestamp:      ar.Timestamp.Format(time.RFC3339),
		RateLimited:    ar.RateLimited,
	}

	// Format user ID if present
	if ar.UserID != nil {
		userID := ar.UserID.String()
		response.UserID = &userID
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
	return ar.ResponseTimeMs > thresholdMs
}

// GetEndpointGroup returns a simplified endpoint group for analytics
// e.g., "/api/images/123" -> "/api/images/{id}"
func (ar *APIRequest) GetEndpointGroup() string {
	// This would contain logic to group similar endpoints
	// For now, return the endpoint as-is
	return ar.Endpoint
}