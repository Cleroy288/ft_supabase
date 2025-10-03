package ft_supabase

// Auth API endpoint path constants.
const (
	// AuthBasePath is the base path for all auth endpoints.
	AuthBasePath = "/auth/v1"

	// SignupPath is the endpoint path for user registration.
	SignupPath = "/auth/v1/signup"

	// LoginPath is the endpoint path for user login with password grant.
	LoginPath = "/auth/v1/token?grant_type=password"

	// UserPath is the endpoint path for user operations.
	UserPath = "/auth/v1/user"

	// LogoutPath is the endpoint path for user logout.
	LogoutPath = "/auth/v1/logout"

	// RefreshTokenPath is the endpoint path for token refresh.
	RefreshTokenPath = "/auth/v1/token?grant_type=refresh_token"

	// ResetPasswordPath is the endpoint path for password recovery.
	ResetPasswordPath = "/auth/v1/recover"

	// UpdateUserPath is the endpoint path for user updates.
	UpdateUserPath = "/auth/v1/user"

	// DeleteUserPath is the endpoint path for user deletion (admin endpoint).
	DeleteUserPath = "/auth/v1/admin/users"
)
