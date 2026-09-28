package cfg

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/types"

	"github.com/joho/godotenv"
)

type Config struct {
	Site       SiteConfig
	Database   DatabaseConfig
	AliyunOSS  AliyunOSSConfig
	AliyunSMQ  AliyunSMQConfig
	Ably       AblyConfig
	WhatsApp   WhatsAppConfig
	Commerce   CommerceConfig
	Cloudflare Cloudflare
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
	HTTPRequestWebsiteKey        string // to retrieve user's customer website
	HTTPRequestIdKey             string // to retrieve per-request correlation id from context, used to correlate access/error logs
	HTTPHeaderUserAccessTokenKey string // to retrieve user access token string from context, used in authenticate\\
	SessionExpirySeconds         int
	GoogleMapAPIKey              string
	GoogleDarkMapID              string
	GoogleLightMapID             string
	GlobalKeys                   *helper.CryptoKeys
	GlobalKeyRing                *helper.CryptoKeyRing
}

type AliyunOSSConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
}

type AliyunSMQConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	QueueName       string
	MaxDequeueCount int
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

type CommerceConfig struct {
	WebsiteDomain string
}

type Cloudflare struct {
	TurnstileSecretKey string
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
		environment := types.Environment(os.Getenv("ENVIRONMENT"))
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
				Environment:                  environment,
				HTTPRequestUserKey:           "HTTP_REQUEST_USER",
				HTTPRequestItemKey:           "HTTP_REQUEST_ITEM",
				HTTPRequestIdKey:             "HTTP_REQUEST_ID",
				HTTPRequestWebsiteKey:        "HTTP_REQUEST_WEBSITE",
				HTTPHeaderUserAccessTokenKey: "x-user-access-token",
				GoogleMapAPIKey:              os.Getenv("GOOGLE_MAP_APIKEY"),
				GoogleDarkMapID:              os.Getenv("GOOGLE_DARK_MAP_ID"),
				GoogleLightMapID:             os.Getenv("GOOGLE_LIGHT_MAP_ID"),
				SessionExpirySeconds:         14 * 24 * 60 * 60, // 14 days
				GlobalKeys:                   loadGlobalKeys(environment),
				GlobalKeyRing:                loadGlobalKeyRing(environment),
			},
			AliyunOSS: AliyunOSSConfig{
				Endpoint:        os.Getenv("ALIYUN_OSS_ENDPOINT"),
				AccessKeyID:     os.Getenv("ALIYUN_OSS_ACCESS_KEY_ID"),
				AccessKeySecret: os.Getenv("ALIYUN_OSS_ACCESS_KEY_SECRET"),
				BucketName:      os.Getenv("ALIYUN_OSS_BUCKET_NAME"),
			},
			AliyunSMQ: AliyunSMQConfig{
				Endpoint:        os.Getenv("ALIYUN_SMQ_ENDPOINT"),
				AccessKeyID:     os.Getenv("ALIYUN_SMQ_ACCESS_KEY_ID"),
				AccessKeySecret: os.Getenv("ALIYUN_SMQ_ACCESS_KEY_SECRET"),
				QueueName:       os.Getenv("ALIYUN_SMQ_QUEUE_NAME"),
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
			Commerce: CommerceConfig{
				WebsiteDomain: os.Getenv("COMMERCE_WEBSITE_DOMAIN"),
			},
			Cloudflare: Cloudflare{
				TurnstileSecretKey: os.Getenv("TURNSTILE_SECRET_KEY"),
			},
		}
	})
	// set max dequeue count in SMQ (max number of deliveries)
	smqMaxDequeueCount, err := strconv.Atoi(os.Getenv("ALIYUN_SMQ_MAX_DEQUEUE_COUNT"))
	if err != nil {
		configInstance.AliyunSMQ.MaxDequeueCount = 3
	} else {
		configInstance.AliyunSMQ.MaxDequeueCount = smqMaxDequeueCount
	}
	return configInstance
}

func loadGlobalKeys(environment types.Environment) *helper.CryptoKeys {
	version := int32(1)
	if versionValue := strings.TrimSpace(os.Getenv("ENCRYPTION_CURRENT_VERSION")); versionValue != "" {
		parsedVersion, parseErr := strconv.ParseInt(versionValue, 10, 32)
		if parseErr != nil || parsedVersion < 1 {
			panic("ENCRYPTION_CURRENT_VERSION must be a positive integer")
		}
		version = int32(parsedVersion)
	}
	masterKeyValue := versionedEnvValue("ENCRYPTION_MASTER_KEY", version)
	saltValue := versionedEnvValue("ENCRYPTION_SALT", version)
	if masterKeyValue == "" || saltValue == "" {
		panic(fmt.Sprintf("encryption key material for version %d is missing", version))
	}
	masterKey, err := base64.StdEncoding.DecodeString(masterKeyValue)
	if err != nil {
		panic(fmt.Sprintf("invalid ENCRYPTION_MASTER_KEY: %v", err))
	}
	salt, err := base64.StdEncoding.DecodeString(saltValue)
	if err != nil {
		panic(fmt.Sprintf("invalid ENCRYPTION_SALT: %v", err))
	}
	keys, err := helper.DeriveKeys(masterKey, salt, version, environment)
	if err != nil {
		panic(fmt.Sprintf("invalid encryption configuration: %v", err))
	}
	return keys
}

func versionedEnvValue(prefix string, version int32) string {
	if value := strings.TrimSpace(os.Getenv(fmt.Sprintf("%s_V%d", prefix, version))); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(prefix))
}

func loadGlobalKeyRing(environment types.Environment) *helper.CryptoKeyRing {
	current := loadGlobalKeys(environment)
	keys := []*helper.CryptoKeys{}
	versions := strings.TrimSpace(os.Getenv("ENCRYPTION_KEY_VERSIONS"))
	if versions != "" {
		for _, value := range strings.Split(versions, ",") {
			version, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
			if err != nil || version < 1 || int32(version) == current.Version {
				continue
			}
			masterKey, masterErr := decodeKeyEnv(fmt.Sprintf("ENCRYPTION_MASTER_KEY_V%d", version), 32)
			salt, saltErr := decodeKeyEnv(fmt.Sprintf("ENCRYPTION_SALT_V%d", version), 32)
			if masterErr != nil || saltErr != nil {
				panic(fmt.Sprintf("invalid encryption key version %d", version))
			}
			key, deriveErr := helper.DeriveKeys(masterKey, salt, int32(version), environment)
			if deriveErr != nil {
				panic(fmt.Sprintf("invalid encryption key version %d: %v", version, deriveErr))
			}
			keys = append(keys, key)
		}
	}
	keyRing, err := helper.NewCryptoKeyRing(current, keys...)
	if err != nil {
		panic(fmt.Sprintf("invalid encryption key ring: %v", err))
	}
	return keyRing
}

func decodeKeyEnv(name string, minimumBytes int) ([]byte, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return nil, fmt.Errorf("%s is missing", name)
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(decoded) < minimumBytes {
		return nil, fmt.Errorf("%s is invalid", name)
	}
	return decoded, nil
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
