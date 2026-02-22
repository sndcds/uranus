package app

import (
	"fmt"
)

// TODO: Review code

// Config holds database configuration details
type Config struct {
	Verbose                 bool     `json:"verbose"`
	DevMode                 bool     `json:"dev_mode"`
	Port                    int      `json:"port"`
	BaseApiUrl              string   `json:"base_api_url"`
	ICSDomain               string   `json:"ics_domain"`
	UseRouterMiddleware     bool     `json:"use_router_middleware"`
	SupportedLanguages      []string `json:"supported_languages"`
	DbHost                  string   `json:"db_host"`
	DbPort                  int      `json:"db_port"`
	DbUser                  string   `json:"db_user"`
	DbPassword              string   `json:"db_password"`
	DbName                  string   `json:"db_name"`
	DbSchema                string   `json:"db_schema"`
	SSLMode                 string   `json:"ssl_mode"`
	AllowOrigins            []string `json:"allow_origins"`
	ProfileImageDir         string   `json:"profile_image_dir"`
	ProfileImageQuality     float32  `json:"profile_image_quality"`
	PlutoImageMaxFileSize   int      `json:"pluto_image_max_file_size"`
	PlutoImageMaxPx         int      `json:"pluto_image_max_px"`
	PlutoVerbose            bool     `json:"pluto_verbose"`
	PlutoImageDir           string   `json:"pluto_image_dir"`
	PlutoCacheDir           string   `json:"pluto_cache_dir"`
	JwtSecret               string   `json:"jwt_secret"`
	SecretKey               string   `json:"secret_key"`
	AuthTokenExpirationTime int      `json:"auth_token_expiration_time"`
	AuthSmtpHost            string   `json:"auth_smtp_host"`
	AuthSmtpPort            int      `json:"auth_smtp_port"`
	AuthSmtpLogin           string   `json:"auth_smtp_login"`
	AuthSmtpPassword        string   `json:"auth_smtp_password"`
	AuthReplyEmailAddress   string   `json:"auth_reply_email_address"`
	AuthResetPasswordUrl    string   `json:"auth_reset_password_url"`
	SmtpHost                string   `json:"smtp_host"`
	SmtpPort                int      `json:"smtp_port"`
	SmtpLogin               string   `json:"smtp_login"`
	SmtpPassword            string   `json:"smtp_password"`
}

func (config Config) Print() {
	fmt.Println("app Config")
	fmt.Printf("  verbose: %t\n", config.Verbose)
	fmt.Printf("  dev_mode: %t\n", config.DevMode)
	fmt.Printf("  port: %d\n", config.Port)
	fmt.Printf("  supported_languages: %v\n", config.SupportedLanguages)
	fmt.Printf("  base_api_url: %s\n", config.BaseApiUrl)
	fmt.Printf("  ics_domain: %s\n", config.ICSDomain)
	fmt.Printf("  use_router_middleware: %t\n", config.UseRouterMiddleware)
	fmt.Printf("  db_host: %s\n", config.DbHost)
	fmt.Printf("  db_port: %d\n", config.DbPort)
	fmt.Printf("  db_user: %s\n", config.DbUser)
	fmt.Printf("  db_name: %s\n", config.DbName)
	fmt.Printf("  db_schema: %s\n", config.DbSchema)
	fmt.Printf("  ssl_mode: %s\n", config.SSLMode)
	fmt.Printf("  allow_origins: %v\n", config.AllowOrigins)
	fmt.Printf("  pluto_verbose: %t\n", config.PlutoVerbose)
	fmt.Printf("  pluto_image_dir: %s\n", config.PlutoImageDir)
	fmt.Printf("  pluto_cache_dir: %s\n", config.PlutoCacheDir)
	if config.JwtSecret != "" {
		fmt.Printf("  jwt_secret: [REDACTED] (%d bytes)\n", len(config.JwtSecret))
	} else {
		fmt.Printf("  jwt_secret: Doesn't exist\n")
	}
	fmt.Printf("  auth_token_expiration_time: %d seconds\n", config.AuthTokenExpirationTime)
}
