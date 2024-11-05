package models

type URLStore struct {
	UUID          int    `json:"uuid,omitempty" db:"-"`
	CorrelationID string `json:"correlation_id,omitempty" db:"-"`
	ShortURL      string `json:"short_url" db:"short_url"`
	OriginalURL   string `json:"original_url" db:"original_url"`
	DeletedFlag   bool   `json:"-" db:"is_deleted"`
}
