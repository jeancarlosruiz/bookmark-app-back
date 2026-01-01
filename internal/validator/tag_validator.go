package validator

type CreateTag struct {
	Title string `json:"title" validate:"required,min=1,max=100"`
}

type UpdateTag struct {
	Title string `json:"title" validate:"required,min=1,max=100"`
}
