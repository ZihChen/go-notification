package config

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/viper"
)

// Config 應用程序配置
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	AWS      AWSConfig
	Tracing  TracingConfig
	Events   EventsConfig
	Auth     AuthConfig
}

// AppConfig 應用程序基本配置
type AppConfig struct {
	Name  string
	Env   string
	Port  int
	Debug bool
}

// ServerConfig HTTP服務器配置
type ServerConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig 資料庫配置
type DatabaseConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	Options     string
	MaxIdle     int
	MaxOpen     int
	MaxLifetime time.Duration
	MaxIdleTime time.Duration
}

// RedisConfig Redis配置
type RedisConfig struct {
	Domain       string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
	IdleTimeout  time.Duration
	MaxConnAge   time.Duration
}

// AWSConfig AWS配置
type AWSConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
	KinesisStream   string
	DynamoDBTable   string
	PartitionKey    string
	SortKey         string
}

// TracingConfig 追踪配置
type TracingConfig struct {
	Endpoint   string
	APIKey     string
	StreamName string
}

// EventsConfig 事件配置
type EventsConfig struct {
	IdentityMerchantSync    string
	IdentityPlayerSync      string
	IdentityManagerSync     string
	IdentityTagSync         string
	IdentityPlayerLevelSync string
	IdentityPlayerTagsSync  string
}

// AuthConfig API認證配置
type AuthConfig struct {
	Enabled        bool
	APIKeys        map[string]string
	HeaderKey      string
	EncryptionType string
}

// LoadConfig 加載配置
func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	config := &Config{
		App: AppConfig{
			Name:  viper.GetString("APP_NAME"),
			Env:   viper.GetString("APP_ENV"),
			Port:  viper.GetInt("APP_PORT"),
			Debug: viper.GetBool("APP_DEBUG"),
		},
		Server: ServerConfig{
			ReadTimeout:  getTimeWithDefault("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getTimeWithDefault("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getTimeWithDefault("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			Host:        viper.GetString("DB_HOST"),
			Port:        viper.GetInt("DB_PORT"),
			User:        viper.GetString("DB_USER"),
			Password:    viper.GetString("DB_PASSWORD"),
			DBName:      viper.GetString("DB_NAME"),
			Options:     viper.GetString("DB_OPTIONS"),
			MaxIdle:     getIntWithDefault("DB_MAX_IDLE", 25),
			MaxOpen:     getIntWithDefault("DB_MAX_OPEN", 100),
			MaxLifetime: getTimeWithDefault("DB_MAX_LIFETIME", 1*time.Hour),
			MaxIdleTime: getTimeWithDefault("DB_MAX_IDLE_TIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Domain:       viper.GetString("REDIS_DOMAIN"),
			Port:         viper.GetInt("REDIS_PORT"),
			Password:     viper.GetString("REDIS_PWD"),
			DB:           viper.GetInt("REDIS_DB"),
			PoolSize:     getIntWithDefault("REDIS_POOL_SIZE", 20),
			MinIdleConns: getIntWithDefault("REDIS_MIN_IDLE_CONNS", 5),
			MaxRetries:   getIntWithDefault("REDIS_MAX_RETRIES", 3),
			DialTimeout:  getTimeWithDefault("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getTimeWithDefault("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getTimeWithDefault("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolTimeout:  getTimeWithDefault("REDIS_POOL_TIMEOUT", 4*time.Second),
			IdleTimeout:  getTimeWithDefault("REDIS_IDLE_TIMEOUT", 5*time.Minute),
			MaxConnAge:   getTimeWithDefault("REDIS_MAX_CONN_AGE", 30*time.Minute),
		},
		AWS: AWSConfig{
			AccessKeyID:     viper.GetString("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: viper.GetString("AWS_SECRET_ACCESS_KEY"),
			SessionToken:    viper.GetString("AWS_SESSION_TOKEN"),
			Region:          viper.GetString("AWS_REGION"),
			KinesisStream:   viper.GetString("KINESIS_STREAM_NAME"),
			DynamoDBTable:   viper.GetString("DYNAMODB_TABLE"),
			PartitionKey:    viper.GetString("DYNAMODB_PARTITION_KEY"),
			SortKey:         viper.GetString("DYNAMODB_SORT_KEY"),
		},
		Tracing: TracingConfig{
			Endpoint:   viper.GetString("OPENOBSERVE_TRACE_API_ENDPOINT"),
			APIKey:     viper.GetString("OPENOBSERVE_TRACE_API_KEY"),
			StreamName: viper.GetString("OPENOBSERVE_TRACE_STREAM_NAME"),
		},
		Events: EventsConfig{
			IdentityMerchantSync:    viper.GetString("EVENT_IDENTITY_MERCHANT_SYNC"),
			IdentityPlayerSync:      viper.GetString("EVENT_IDENTITY_PLAYER_SYNC"),
			IdentityManagerSync:     viper.GetString("EVENT_IDENTITY_MANAGER_SYNC"),
			IdentityTagSync:         viper.GetString("EVENT_IDENTITY_TAG_SYNC"),
			IdentityPlayerLevelSync: viper.GetString("EVENT_IDENTITY_PLAYER_LEVEL_SYNC"),
			IdentityPlayerTagsSync:  viper.GetString("EVENT_IDENTITY_PLAYER_TAGS_SYNC"),
		},
		Auth: AuthConfig{
			Enabled:        viper.GetBool("AUTH_ENABLED"),
			APIKeys:        parseAPIKeyMap(viper.GetString("AUTH_API_KEYS")),
			HeaderKey:      viper.GetString("AUTH_HEADER_KEY"),
			EncryptionType: viper.GetString("AUTH_ENCRYPTION_TYPE"),
		},
	}

	return config, nil
}

// LoadAWSConfig 加載AWS配置
func (c *Config) LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	var opts []func(*awsconfig.LoadOptions) error

	opts = append(
		opts,
		awsconfig.WithRegion(c.AWS.Region),
		awsconfig.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = 3               // 最大重試次數
				o.MaxBackoff = 10 * time.Second // 最大重試間隔
			})
		}),
	)

	if c.App.Env == "local" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     c.AWS.AccessKeyID,
					SecretAccessKey: c.AWS.SecretAccessKey,
					SessionToken:    c.AWS.SessionToken,
				}, nil
			}),
		))
	}
	return awsconfig.LoadDefaultConfig(ctx, opts...)
}

// parseAPIKeyMap 解析JSON格式的API Key映射字符串
func parseAPIKeyMap(apiKeysString string) map[string]string {
	if apiKeysString == "" {
		return map[string]string{}
	}

	keyMap := make(map[string]string)
	err := json.Unmarshal([]byte(apiKeysString), &keyMap)
	if err != nil {
		// 如果JSON解析失败，返回空map
		return map[string]string{}
	}

	return keyMap
}

// Helper functions for default values
func getTimeWithDefault(key string, defaultValue time.Duration) time.Duration {
	if value := viper.GetDuration(key); value != 0 {
		return value
	}
	return defaultValue
}

func getIntWithDefault(key string, defaultValue int) int {
	if value := viper.GetInt(key); value != 0 {
		return value
	}
	return defaultValue
}

func getBoolWithDefault(key string, defaultValue bool) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return defaultValue
}
