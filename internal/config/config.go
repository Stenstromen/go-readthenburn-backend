package config

import "os"

type Config struct {
	DBDriver           string
	DBUser             string
	DBPass             string
	DBName             string
	DBHost             string
	SecretKey          string
	AuthHeaderPassword string
	CorsHeader         string
}

func LoadConfig() *Config {
	return &Config{
		DBDriver:           "mysql",
		DBUser:             os.Getenv("MYSQL_USERNAME"),
		DBPass:             os.Getenv("MYSQL_PASSWORD"),
		DBName:             os.Getenv("MYSQL_DATABASE"),
		DBHost:             os.Getenv("MYSQL_HOSTNAME"),
		SecretKey:          os.Getenv("SECRET_KEY"),
		AuthHeaderPassword: os.Getenv("AUTHHEADER_PASSWORD"),
		CorsHeader:         os.Getenv("CORS_HEADER"),
	}
}
