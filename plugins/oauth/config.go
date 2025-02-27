package oauth

import "golang.org/x/oauth2"

// ClaimRule represents a rule for checking a claim value
type ClaimRule struct {
	// Claim defines the path to the value in JWT or user data (e.g., "email", "groups[0]", "org.name")
	Claim string `yaml:"claim"`

	// Operation defines the comparison operation ("eq", "ne", "contains", "regex", "exists")
	Operation string `yaml:"operation"`

	// Value is the expected value for comparison
	Value string `yaml:"value"`
}

// AuthorizationRule defines an authorization rule for a method or group of methods
type AuthorizationRule struct {
	// Methods defines the list of methods to which the rule applies
	Methods []string `yaml:"methods"`

	// AllowPublic allows public access without a token
	AllowPublic bool `yaml:"allow_public"`

	// RequireAllClaims determines if all ClaimRules must be true (AND)
	// If false, one true rule is sufficient (OR)
	RequireAllClaims bool `yaml:"require_all_claims"`

	// ClaimRules list of claim validation rules
	ClaimRules []ClaimRule `yaml:"claim_rules"`
}

// Config represents OAuth plugin configuration
type Config struct {
	// Provider specifies the OAuth provider ("google", "github", "auth0", "keycloak", "okta")
	Provider string `yaml:"provider"`

	// ProviderAuthURL specifies oauth2.Endpoint AuthURL if Provider is unknown
	ProviderAuthURL string `yaml:"provider_auth_url"`

	// ProviderTokenURL specifies oauth2.Endpoint TokenURL if Provider is unknown
	ProviderTokenURL string `yaml:"provider_token_url"`

	// ClientID is the OAuth Client ID
	ClientID string `yaml:"client_id"`

	// ClientSecret is the OAuth Client Secret
	ClientSecret string `yaml:"client_secret"`

	// RedirectURL for OAuth flow
	RedirectURL string `yaml:"redirect_url"`

	// Scopes defines required access scopes
	Scopes []string `yaml:"scopes"`

	// MethodScopes defines required scopes for specific methods
	MethodScopes map[string][]string `yaml:"method_scopes"`

	// TokenHeader defines the header name for the token (default: "Authorization")
	TokenHeader string `yaml:"token_header"`

	// AuthURL is the gateway's authorization endpoint path (default: "/oauth/authorize")
	AuthURL string `yaml:"auth_url"`

	// CallbackURL is the gateway's callback endpoint path (default: "/oauth/callback")
	CallbackURL string `yaml:"callback_url"`

	// UserInfoURL is the endpoint for retrieving user information (required for Auth0)
	UserInfoURL string `yaml:"user_info_url"`

	// IntrospectionURL is the token introspection endpoint (required for Keycloak and Okta)
	IntrospectionURL string `yaml:"introspection_url"`

	// AuthorizationRules defines authorization rules for methods
	AuthorizationRules []AuthorizationRule `yaml:"authorization_rules"`
}

func (c Config) Tag() string {
	return "oauth"
}

func (c Config) Doc() string {
	return docString
}

// GetOAuthConfig returns oauth2.Config for the specified provider
func (c Config) GetOAuthConfig() *oauth2.Config {
	var endpoint oauth2.Endpoint

	switch c.Provider {
	case "google":
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		}
	case "github":
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://github.com/login/oauth/authorize",
			TokenURL: "https://github.com/login/oauth/access_token",
		}
	default:
		endpoint = oauth2.Endpoint{
			AuthURL:  c.ProviderAuthURL,
			TokenURL: c.ProviderTokenURL,
		}
		// Add other providers as needed
	}

	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURL,
		Scopes:       c.Scopes,
		Endpoint:     endpoint,
	}
}

// WithDefaults sets default values for the config fields
func (c *Config) WithDefaults() {
	if c.TokenHeader == "" {
		c.TokenHeader = "Authorization"
	}
	if c.AuthURL == "" {
		c.AuthURL = "/oauth/authorize"
	}
	if c.CallbackURL == "" {
		c.CallbackURL = "/oauth/callback"
	}
}
