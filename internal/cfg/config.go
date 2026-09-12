package cfg

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"

	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/joho/godotenv"
)

type Config struct {
	Site              SiteConfig
	Database          DatabaseConfig
	AliyunOSS         AliyunOSSConfig
	AliyunSMQ         AliyunSMQConfig
	AliyunEventBridge AliyunEventBridgeConfig
	Ably              AblyConfig
	WhatsApp          WhatsAppConfig
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
	HTTPRequestUserKey           string // to retrieve user object from context, used in controller.registerRoute
	HTTPRequestItemKey           string // to retrieve request object from context, used in bindRequest
	HTTPRequestIdKey             string // to retrieve per-request correlation id from context, used to correlate access/error logs
	HTTPHeaderUserAccessTokenKey string // to retrieve user access token string from context, used in authenticate
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

type AliyunEventBridgeConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	EventBusName    string
	Source          string
}

type AblyConfig struct {
	APIKey string
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
				HTTPRequestUserKey:           "HTTP_REQUEST_USER",
				HTTPRequestItemKey:           "HTTP_REQUEST_ITEM",
				HTTPRequestIdKey:             "HTTP_REQUEST_ID",
				HTTPHeaderUserAccessTokenKey: "x-user-access-token",
				GoogleMapAPIKey:              os.Getenv("GOOGLE_MAP_APIKEY"),
				SessionExpirySeconds:         14 * 24 * 60 * 60, // 14 days
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
			AliyunEventBridge: AliyunEventBridgeConfig{
				Endpoint:        os.Getenv("ALIYUN_EVENTBRIDGE_ENDPOINT"),
				AccessKeyID:     os.Getenv("ALIYUN_EVENTBRIDGE_ACCESS_KEY_ID"),
				AccessKeySecret: os.Getenv("ALIYUN_EVENTBRIDGE_ACCESS_KEY_SECRET"),
				EventBusName:    os.Getenv("ALIYUN_EVENTBRIDGE_EVENT_BUS_NAME"),
				Source:          os.Getenv("ALIYUN_EVENTBRIDGE_SOURCE"),
			},
			Ably: AblyConfig{
				APIKey: os.Getenv("ABLY_API_KEY"),
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

func (config *DatabaseConfig) URL() string {
	return config.databaseURL(config.Name)
}

func (config *DatabaseConfig) MigrateDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		config.Host, config.Port, config.User, config.Password, config.MigrationName, config.SSLMode)
}

func (config *DatabaseConfig) MigrateURL() string {
	return config.databaseURL(config.MigrationName)
}

func (config *DatabaseConfig) databaseURL(databaseName string) string {
	query := url.Values{}
	query.Set("sslmode", config.SSLMode)
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(config.User, config.Password),
		Host:     net.JoinHostPort(config.Host, config.Port),
		Path:     "/" + databaseName,
		RawQuery: query.Encode(),
	}).String()
}
