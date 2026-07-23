package models

import "time"

type Category struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Slug        string     `json:"slug"`
	ParentID    *string    `json:"parent_id,omitempty"`
	IsActive    bool       `json:"is_active"`
	SortOrder   int        `json:"sort_order"`
	Children    []Category `json:"children,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
