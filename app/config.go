package app

import (
	"fmt"
)

// Config holds database configuration details
type Config struct {
	Verbose             bool     `json:"verbose"`
	DevMode             bool     `json:"dev_mode"`
	Port                int      `json:"port"`
	BaseApiUrl          string   `json:"base_api_url"`
	UseRouterMiddleware bool     `json:"use_router_middleware"`
	DbHost              string   `json:"db_host"`
	DbPort              int      `json:"db_port"`
	DbUser              string   `json:"db_user"`
	DbPassword          string   `json:"db_password"`
	DbName              string   `json:"db_name"`
	DbSchema            string   `json:"db_schema"`
	SSLMode             string   `json:"ssl_mode"`
	AllowOrigins        []string `json:"allow_origins"`
	PlutoVerbose        bool     `json:"pluto_verbose"`
	PlutoImageDir       string   `json:"pluto_image_dir"`
	PlutoCacheDir       string   `json:"pluto_cache_dir"`
}

func (config Config) Print() {
	fmt.Println("app Config")
	fmt.Printf("  verbose: %t\n", config.Verbose)
	fmt.Printf("  dev_mode: %t\n", config.DevMode)
	fmt.Printf("  port: %d\n", config.Port)
	fmt.Printf("  base_api_url: %s\n", config.BaseApiUrl)
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
}
