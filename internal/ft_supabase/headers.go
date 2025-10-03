package ft_supabase

// HTTP header constants for Supabase API requests.
const (
	// HeaderContentType is the Content-Type header key.
	HeaderContentType = "Content-Type"

	// HeaderAPIKey is the Supabase API key header key.
	HeaderAPIKey = "apikey"

	// HeaderAuthorization is the Authorization header key.
	HeaderAuthorization = "Authorization"

	// ContentTypeJSON is the JSON content type value.
	ContentTypeJSON = "application/json"
)

// getDefaultHeaders returns the default headers required for Supabase API requests.
// Returns a map of header key-value pairs with Content-Type and apikey.
func (s *Service) getDefaultHeaders() map[string]string {
	// return default headers
	return map[string]string{
		HeaderContentType: ContentTypeJSON,
		HeaderAPIKey:      s.AnonKey,
	}
}

// getAuthHeaders returns headers with Bearer token authentication.
// token is the JWT access token.
// Returns a map of header key-value pairs with Content-Type and Authorization.
func (s *Service) getAuthHeaders(token string) map[string]string {
	// return authenticated headers
	return map[string]string{
		HeaderContentType:   ContentTypeJSON,
		HeaderAuthorization: "Bearer " + token,
		HeaderAPIKey:        s.AnonKey,
	}
}

// getServiceHeaders returns headers with service role key for admin operations.
// Returns a map of header key-value pairs with Content-Type and service role apikey.
func (s *Service) getServiceHeaders() map[string]string {
	// return service role headers
	return map[string]string{
		HeaderContentType:   ContentTypeJSON,
		HeaderAuthorization: "Bearer " + s.ServiceKey,
		HeaderAPIKey:        s.ServiceKey,
	}
}
