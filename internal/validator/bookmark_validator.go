package validator

type CreateBookmark struct {
	Title  string   `validate:"required"`
	Url    string   `validate:"required"`
	UserID string   `validate:"required"`
	Tags   []string `validate:"omitempty,dive,min=1"`
}
