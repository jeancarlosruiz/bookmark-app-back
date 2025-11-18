package validator

type CreateBookmark struct {
	Title  string   `validate:"required"`
	Url    string   `validate:"required"`
	UserID string   `validate:"required"`
	Tags   []string `validate:"omitempty,dive,min=1"`
}

type UpdateBookmark struct {
	Title       *string  `json:"title" validate:"omitempty,min=1,max=200"`
	Url         *string  `json:"utl" validate:"omitempty,url,http_url"`
	Description *string  `json:"description" validate:"omitempty,max=1000"`
	Favicon     *string  `json:"favicon" validate:"omitempty,url"`
	Pinned      *bool    `json:"pinned"`
	IsArchived  *bool    `json:"is_archived"`
	Tags        []string `json:"tags" validate:"omitempty,dive,min=1,max=50"`
}
