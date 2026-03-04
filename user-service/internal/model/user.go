package model

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required" example:"John Doe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required" example:"secret123"`
}

type LoginResponse struct {
	Token string       `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string    `json:"name" example:"John Doe"`
	Email     string    `json:"email" example:"john@example.com"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProfileDetailResponse aggregates all available user profile columns into a single structured response.
type ProfileDetailResponse struct {
	ID              uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name            string    `json:"name" example:"John Doe"`
	Email           string    `json:"email" example:"john@example.com"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	MemberSinceDays int       `json:"member_since_days" example:"365"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}

type MessageResponse struct {
	Message string `json:"message" example:"success"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// ToDetailResponse converts a User to a ProfileDetailResponse with computed metadata.
func (u *User) ToDetailResponse() ProfileDetailResponse {
	days := int(math.Floor(time.Since(u.CreatedAt).Hours() / 24))
	return ProfileDetailResponse{
		ID:              u.ID,
		Name:            u.Name,
		Email:           u.Email,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
		MemberSinceDays: days,
	}
}
