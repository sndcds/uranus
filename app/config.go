package app

import (
	"encoding/json"
	"fmt"
)

// TODO: Review code

// Config holds database configuration details
type Config struct {
	Verbose                     bool     `json:"verbose"`
	DevMode                     bool     `json:"dev_mode"`
	DebugLevel                  int      `json:"debug_level"`
	Port                        int      `json:"port"`
	BaseApiUrl                  string   `json:"base_api_url"`
	IcsDomain                   string   `json:"ics_domain"`
	Frontend                    string   `json:"frontend"`
	UseRouterMiddleware         bool     `json:"use_router_middleware"`
	SupportedLanguages          []string `json:"supported_languages"`
	DbHost                      string   `json:"db_host"`
	DbPort                      int      `json:"db_port"`
	DbUser                      string   `json:"db_user"`
	DbPassword                  string   `json:"db_password"`
	DbName                      string   `json:"db_name"`
	DbSchema                    string   `json:"db_schema"`
	SSLMode                     string   `json:"ssl_mode"`
	AllowOrigins                []string `json:"allow_origins"`
	ProfileImageDir             string   `json:"profile_image_dir"`
	ProfileImageQuality         float32  `json:"profile_image_quality"`
	PlutoImageMaxFileSize       int      `json:"pluto_image_max_file_size"`
	PlutoImageMaxPx             int      `json:"pluto_image_max_px"`
	PlutoVerbose                bool     `json:"pluto_verbose"`
	PlutoImageDir               string   `json:"pluto_image_dir"`
	PlutoCacheDir               string   `json:"pluto_cache_dir"`
	JwtSecret                   string   `json:"jwt_secret"`
	SecretKey                   string   `json:"secret_key"`
	AuthTokenExpirationTime     int      `json:"auth_token_expiration_time"`
	AuthSmtpHost                string   `json:"auth_smtp_host"`
	AuthSmtpPort                int      `json:"auth_smtp_port"`
	AuthSmtpLogin               string   `json:"auth_smtp_login"`
	AuthSmtpPassword            string   `json:"auth_smtp_password"`
	AuthReplyEmail              string   `json:"auth_reply_email"`
	AuthResetPasswordUrl        string   `json:"auth_reset_password_url"`
	InvitationExpirationMinutes int      `json:"invitation_expiration_minutes"`
}

func (config Config) Print() {
	fmt.Println("Uranus Config")

	b, err := json.MarshalIndent(config, "  ", "  ")
	if err != nil {
		fmt.Println("  Error printing config:", err)
		return
	}

	fmt.Println(string(b))
}

func DefaultConfig() Config {
	return Config{
		Verbose:                     false,
		PlutoVerbose:                false,
		DevMode:                     false,
		DebugLevel:                  1,
		Port:                        9090,
		BaseApiUrl:                  "http://localhost:9090",
		UseRouterMiddleware:         true,
		SupportedLanguages:          []string{"en", "de", "da"},
		DbHost:                      "localhost",
		DbPort:                      5432,
		SSLMode:                     "disable",
		ProfileImageQuality:         0.8,
		PlutoImageMaxFileSize:       5_000_000,
		PlutoImageMaxPx:             1920,
		AuthTokenExpirationTime:     360,
		InvitationExpirationMinutes: 60,
	}
}
