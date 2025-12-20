package models

type User struct {
	ID string `gorm:"primaryKey;column:id;type:text"` // UUID generado por Better Auth
}

// TableName especifica la tabla en el schema public
func (User) TableName() string {
	return "user" // Apunta a public.user (tabla de Better Auth)
}
