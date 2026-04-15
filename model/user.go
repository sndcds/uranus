package model

type User struct {
	Uuid         string  `json:"uuid"`
	Email        string  `json:"email"`
	PasswordHash *string `json:"password_hash"`
	Username     *string `json:"username"`
	DisplayName  *string `json:"display_name"`
	FirstName    *string `json:"first_name"`
	LastName     *string `json:"last_name"`
	Locale       *string `json:"locale"`
	Theme        *string `json:"theme"`
	IsActive     bool    `json:"is_active"`
	AvatarUrl    *string `json:"avatar_url"`
}

type UserCredentials struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserProfileResponse struct {
	UserUUID    string  `json:"user_uuid"`
	Email       string  `json:"email"`
	Username    *string `json:"username"`
	DisplayName *string `json:"display_name"`
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	Locale      *string `json:"locale"`
	Theme       *string `json:"theme"`
	AvatarUrl   *string `json:"avatar_url"`
}
