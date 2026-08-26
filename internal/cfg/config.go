package cfg

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	Site      SiteConfig
	Database  DatabaseConfig
	AliyunOSS AliyunOSSConfig
	AliyunSMQ AliyunSMQConfig
	WhatsApp  WhatsAppConfig
}

type DatabaseConfig struct {
	User        string
	Password    string
	Host        string
	Port        string
	Name        string
	SSLMode     string
	MigrateName string
}

type SiteConfig struct {
	Port                 string
	Version              string
	Environment          types.Environment
	HTTPRequestOwnerKey  string
	HTTPRequestUserKey   string
	HTTPRequestItemKey   string
	HTTPHeaderAPIKey     string
	LocalTimezone        string
	SessionCookieName    string
	SessionExpirySeconds int
	PortalOrigin         string
	GoogleMapAPIKey      string
	GlobalKeys           *helper.CryptoKeys
}

type AliyunOSSConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
}

type AliyunSMQConfig struct {
	Endpoint           string
	AccessKeyID        string
	AccessKeySecret    string
	QueueName          string
	PollingWaitSeconds int64
}

type WhatsAppConfig struct {
	BaseURL              string
	APIVersion           string
	AccessToken          string
	WebhookVerifyToken   string
	MaxDelayStartSeconds float32
}

var configInstance *Config
var onceDefault sync.Once

func Default() *Config {
	onceDefault.Do(func() {
		err := godotenv.Load()
		if err != nil {
			// If fails, try to load from project root (when running tests from subdirectories)
			_ = godotenv.Load("../../.env")
			// In production/Docker, environment variables should already be loaded
		}
		// init port
		port := os.Getenv("PORT")
		if port == "" {
			port = "9000" // Default for FC custom runtime
		}
		// load configs
		configInstance = &Config{
			Database: DatabaseConfig{
				User:        os.Getenv("DB_USER"),
				Password:    os.Getenv("DB_PASSWORD"),
				Host:        os.Getenv("DB_HOST"),
				Port:        os.Getenv("DB_PORT"),
				Name:        os.Getenv("DB_NAME"),
				SSLMode:     os.Getenv("DB_SSLMODE"),
				MigrateName: "wawa_test",
			},
			Site: SiteConfig{
				Port:                 port,
				Version:              os.Getenv("VERSION"),
				Environment:          types.Environment(os.Getenv("ENVIRONMENT")),
				HTTPRequestOwnerKey:  "HTTP_REQUEST_OWNER",
				HTTPRequestUserKey:   "HTTP_REQUEST_USER",
				HTTPRequestItemKey:   "HTTP_REQUEST_ITEM",
				HTTPHeaderAPIKey:     "x-api-key",
				LocalTimezone:        os.Getenv("LOCAL_TIMEZONE"),
				GoogleMapAPIKey:      os.Getenv("GOOGLE_MAP_APIKEY"),
				SessionCookieName:    "wawa_session",
				SessionExpirySeconds: 7 * 24 * 60 * 60,
				PortalOrigin:         os.Getenv("PORTAL_ORIGIN"),
			},
			AliyunOSS: AliyunOSSConfig{
				Endpoint:        os.Getenv("ALIYUN_OSS_ENDPOINT"),
				AccessKeyID:     os.Getenv("ALIYUN_OSS_ACCESS_KEY_ID"),
				AccessKeySecret: os.Getenv("ALIYUN_OSS_ACCESS_KEY_SECRET"),
				BucketName:      os.Getenv("ALIYUN_OSS_BUCKET_NAME"),
			},
			AliyunSMQ: AliyunSMQConfig{
				Endpoint:           os.Getenv("ALIYUN_SMQ_ENDPOINT"),
				AccessKeyID:        os.Getenv("ALIYUN_SMQ_ACCESS_KEY_ID"),
				AccessKeySecret:    os.Getenv("ALIYUN_SMQ_ACCESS_KEY_SECRET"),
				QueueName:          os.Getenv("ALIYUN_SMQ_QUEUE_NAME"),
				PollingWaitSeconds: 15,
			},
			WhatsApp: WhatsAppConfig{
				BaseURL:              "https://graph.facebook.com",
				APIVersion:           "v25.0",
				AccessToken:          os.Getenv("WHATSAPP_ACCESS_TOKEN"),
				WebhookVerifyToken:   os.Getenv("WHATSAPP_WEBHOOK_VERIFY_TOKEN"),
				MaxDelayStartSeconds: 10,
			},
		}
	})
	return configInstance
}

// database related - PostgreSQL connection strings
func (config *DatabaseConfig) EmptyDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s TimeZone=UTC",
		config.Host, config.Port, config.User, config.Password, config.SSLMode)
}

func (config *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC connect_timeout=10",
		config.Host, config.Port, config.User, config.Password, config.Name, config.SSLMode)
}

func (config *DatabaseConfig) MigrateDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		config.Host, config.Port, config.User, config.Password, config.MigrateName, config.SSLMode)
}

func (config *DatabaseConfig) MigrateURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.User, config.Password, config.Host, config.Port, config.MigrateName, config.SSLMode)
}

func (config *DatabaseConfig) CreateMigrationDB() {
	log.Printf("test db name: %s\n", config.MigrateName)
	db, err := gorm.Open(postgres.Open(config.EmptyDSN()), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(err.Error())
	}
	// Terminate all connections to the database before dropping
	err = db.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = '%s' AND pid <> pg_backend_pid()
	`, config.MigrateName)).Error
	if err != nil {
		// It's okay if this fails (database might not exist)
		log.Printf("Warning: Could not terminate connections to %s: %v\n", config.MigrateName, err)
	}
	//drop database first
	err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", config.MigrateName)).Error
	if err != nil {
		panic(err.Error())
	}
	//create database
	err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", config.MigrateName)).Error
	if err != nil {
		panic(err.Error())
	}
	err = db.Exec(fmt.Sprintf("ALTER DATABASE %s SET TIMEZONE TO 'UTC'", config.MigrateName)).Error
	if err != nil {
		panic(err.Error())
	}
}

func (config *DatabaseConfig) DropMigrationDB() {
	// Connect to the main database instead of the migration database to drop it
	db, err := gorm.Open(postgres.Open(config.EmptyDSN()), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		panic(err.Error())
	}
	// Terminate all connections to the migration database before dropping
	err = db.Exec(fmt.Sprintf(`
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = '%s' AND pid <> pg_backend_pid()
	`, config.MigrateName)).Error
	if err != nil {
		// It's okay if this fails (database might not exist)
		log.Printf("Warning: Could not terminate connections to %s: %v\n", config.MigrateName, err)
	}
	//drop the migration database
	err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", config.MigrateName)).Error
	if err != nil {
		panic(err.Error())
	}
}
