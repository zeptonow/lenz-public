package configurator

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/spf13/viper"

	"openreplay/backend/internal/config/common"
	"openreplay/backend/pkg/logger"
)

const (
	VaultStaticFileName  = "static.json"
	VaultDynamicFileName = "dynamic.json"
	VaultConfigPath      = "/vault/secrets/"
)

var (
	viperInstance *viper.Viper
	viperOnce     sync.Once
)

// initViper initializes the Viper instance once and loads config from YAML or Vault.
func initViper(log logger.Logger) *viper.Viper {
	viperOnce.Do(func() {
		ctx := context.Background()
		viperInstance = viper.New()
		viperInstance.AutomaticEnv()

		stage := getEnvOrDefault("STAGE", "local")
		base := getEnvOrDefault("BASE", ".")
		vaultEnabledStr := getEnvOrDefault("VAULT_ENABLED", "false")
		vaultEnabled := vaultEnabledStr == "true" || vaultEnabledStr == "TRUE" || vaultEnabledStr == "True"

		if vaultEnabled {
			log.Info(ctx, "Loading config from Vault at %s", VaultConfigPath)
			viperInstance.AddConfigPath(VaultConfigPath)
			viperInstance.AddConfigPath("./")
			viperInstance.AddConfigPath("../")
			viperInstance.SetConfigType("json")

			// Read static config first
			staticConfigFile := VaultConfigPath + VaultStaticFileName
			viperInstance.SetConfigFile(staticConfigFile)
			if err := viperInstance.ReadInConfig(); err != nil {
				log.Fatal(ctx, "Failed to read static config: %s", err)
			}
			log.Info(ctx, "Loaded static config from %s", staticConfigFile)

			// Merge dynamic config (optional)
			dynamicConfigFile := VaultConfigPath + VaultDynamicFileName
			viperInstance.SetConfigFile(dynamicConfigFile)
			if err := viperInstance.MergeInConfig(); err != nil {
				log.Info(ctx, "Dynamic config not found or disabled: %s", err)
			} else {
				log.Info(ctx, "Merged dynamic config from %s", dynamicConfigFile)
			}
			log.Info(ctx, "Config loaded successfully from Vault")
		} else {
			filePath := path.Join(base, fmt.Sprintf("e.%s.yaml", stage))
			log.Info(ctx, "Loading config from file: %s", filePath)
			viperInstance.SetConfigFile(filePath)

			if err := viperInstance.ReadInConfig(); err != nil {
				log.Fatal(ctx, "Failed to read config: %s", err)
			}
			log.Info(ctx, "Config loaded successfully")
		}
	})
	return viperInstance
}

// Process loads config into the given struct using Viper.
// It replaces the old sethvargo/go-envconfig based loading.
func Process(log logger.Logger, cfg common.Configer) {
	ctx := context.Background()
	v := initViper(log)
	if err := v.Unmarshal(cfg); err != nil {
		log.Fatal(ctx, "Failed to unmarshal config: %s", err)
	}
}

// GetViper returns the initialized Viper instance for direct access if needed.
func GetViper(log logger.Logger) *viper.Viper {
	return initViper(log)
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
