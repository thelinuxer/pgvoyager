package models

import "time"

type SavedQuery struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SQL          string    `json:"sql"`
	ConnectionID string    `json:"connectionId,omitempty"`
	Description  string    `json:"description,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type SavedQueryRequest struct {
	Name         string `json:"name" binding:"required"`
	SQL          string `json:"sql" binding:"required"`
	ConnectionID string `json:"connectionId"`
	Description  string `json:"description"`
}
