package config

import (
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	SAP    SAPConfig    `mapstructure:"sap"`
	DigitalTwin DigitalTwinConfig `mapstructure:"digitalTwin"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// SAPConfig holds SAP connection configuration
type SAPConfig struct {
	BaseURL      string `mapstructure:"baseUrl"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	ClientID     string `mapstructure:"clientId"`
	ClientSecret string `mapstructure:"clientSecret"`
	TokenURL     string `mapstructure:"tokenUrl"`
	Timeout      int    `mapstructure:"timeout"`
	SimulatorMode bool  `mapstructure:"simulatorMode"`
}

// DigitalTwinConfig holds Digital Twin system configuration
type DigitalTwinConfig struct {
	BaseURL string `mapstructure:"baseUrl"`
	APIKey  string `mapstructure:"apiKey"`
	Timeout int    `mapstructure:"timeout"`
}

// Load loads configuration from environment variables and config files
func Load() *Config {
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("sap.timeout", 30)
	viper.SetDefault("sap.simulatorMode", true)
	viper.SetDefault("digitalTwin.timeout", 30)

	// Set environment variable prefix
	viper.SetEnvPrefix("SAP_ADAPTOR")
	viper.AutomaticEnv()

	// Bind environment variables
	viper.BindEnv("server.port", "SAP_ADAPTOR_SERVER_PORT")
	viper.BindEnv("server.host", "SAP_ADAPTOR_SERVER_HOST")
	viper.BindEnv("sap.baseUrl", "SAP_ADAPTOR_SAP_BASE_URL")
	viper.BindEnv("sap.username", "SAP_ADAPTOR_SAP_USERNAME")
	viper.BindEnv("sap.password", "SAP_ADAPTOR_SAP_PASSWORD")
	viper.BindEnv("sap.clientId", "SAP_ADAPTOR_SAP_CLIENT_ID")
	viper.BindEnv("sap.clientSecret", "SAP_ADAPTOR_SAP_CLIENT_SECRET")
	viper.BindEnv("sap.tokenUrl", "SAP_ADAPTOR_SAP_TOKEN_URL")
	viper.BindEnv("sap.timeout", "SAP_ADAPTOR_SAP_TIMEOUT")
	viper.BindEnv("sap.simulatorMode", "SAP_ADAPTOR_SAP_SIMULATOR_MODE")
	viper.BindEnv("digitalTwin.baseUrl", "SAP_ADAPTOR_DIGITAL_TWIN_BASE_URL")
	viper.BindEnv("digitalTwin.apiKey", "SAP_ADAPTOR_DIGITAL_TWIN_API_KEY")
	viper.BindEnv("digitalTwin.timeout", "SAP_ADAPTOR_DIGITAL_TWIN_TIMEOUT")

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	return &config
}
