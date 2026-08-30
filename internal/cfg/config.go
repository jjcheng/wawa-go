package cfg

import (
	"fmt"
	"os"
	"sync"

	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/joho/godotenv"
)

type Config struct {
	Site      SiteConfig
	Database  DatabaseConfig
	AliyunOSS AliyunOSSConfig
	AliyunSMQ AliyunSMQConfig
	WhatsApp  WhatsAppConfig
}

type DatabaseConfig struct {
	User          string
	Password      string
	Host          string
	Port          string
	Name          string
	SSLMode       string
	MigrationName string
}

type SiteConfig struct {
	Port                         string
	Version                      string
	Environment                  types.Environment
	HTTPRequestOwnerKey          string
	HTTPRequestUserKey           string
	HTTPRequestItemKey           string
	HTTPRequestIdKey             string
	HTTPHeaderUserAccessTokenKey string
	LocalTimezone                string
	SessionExpirySeconds         int
	GoogleMapAPIKey              string
	GlobalKeys                   *helper.CryptoKeys
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
	AppID                string
	AppSecret            string
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
				User:          os.Getenv("DB_USER"),
				Password:      os.Getenv("DB_PASSWORD"),
				Host:          os.Getenv("DB_HOST"),
				Port:          os.Getenv("DB_PORT"),
				Name:          os.Getenv("DB_NAME"),
				SSLMode:       os.Getenv("DB_SSLMODE"),
				MigrationName: "wawa_migration",
			},
			Site: SiteConfig{
				Port:                         port,
				Version:                      os.Getenv("VERSION"),
				Environment:                  types.Environment(os.Getenv("ENVIRONMENT")),
				HTTPRequestOwnerKey:          "HTTP_REQUEST_OWNER",
				HTTPRequestUserKey:           "HTTP_REQUEST_USER",
				HTTPRequestItemKey:           "HTTP_REQUEST_ITEM",
				HTTPRequestIdKey:             "HTTP_REQUEST_ID",
				HTTPHeaderUserAccessTokenKey: "x-wawa-user-access-token",
				LocalTimezone:                os.Getenv("LOCAL_TIMEZONE"),
				GoogleMapAPIKey:              os.Getenv("GOOGLE_MAP_APIKEY"),
				SessionExpirySeconds:         7 * 24 * 60 * 60,
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
				APIVersion:           "v26.0",
				AppID:                os.Getenv("META_APP_ID"),
				AppSecret:            os.Getenv("META_APP_SECRET"),
				WebhookVerifyToken:   os.Getenv("WHATSAPP_WEBHOOK_VERIFY_TOKEN"),
				MaxDelayStartSeconds: 10,
			},
		}
	})
	return configInstance
}

func (config *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC connect_timeout=10",
		config.Host, config.Port, config.User, config.Password, config.Name, config.SSLMode)
}

func (config *DatabaseConfig) MigrateDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		config.Host, config.Port, config.User, config.Password, config.MigrationName, config.SSLMode)
}

func (config *DatabaseConfig) MigrateURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.User, config.Password, config.Host, config.Port, config.MigrationName, config.SSLMode)
}
