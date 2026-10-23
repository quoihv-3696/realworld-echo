package dto

type User struct {
	Email    string  `json:"email"`
	Token    string  `json:"token"`
	Username string  `json:"username"`
	Bio      *string `json:"bio"`
	Image    *string `json:"image"`
}

type UserResponse struct {
	User User `json:"user"`
}

type NewUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	User NewUser `json:"user"`
}

type LoginUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	User LoginUser `json:"user"`
}

type UpdateUser struct {
	Email    Optional[string] `json:"email"`
	Username Optional[string] `json:"username"`
	Password Optional[string] `json:"password"`
	Bio      Optional[string] `json:"bio"`
	Image    Optional[string] `json:"image"`
}

type UpdateUserRequest struct {
	User UpdateUser `json:"user"`
}
