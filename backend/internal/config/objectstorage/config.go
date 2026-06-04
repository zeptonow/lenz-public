package objectstorage

// Object storage configuration

type ObjectsConfig struct {
	ServiceName          string `mapstructure:"serviceName"`
	CloudName            string `mapstructure:"cloud"`
	BucketName           string `mapstructure:"bucket"`
	AWSRegion            string `mapstructure:"region"`
	AWSAccessKeyID       string `mapstructure:"accessKeyId"`
	AWSSecretAccessKey   string `mapstructure:"secretAccessKey"`
	AWSEndpoint          string `mapstructure:"endpoint"`
	AWSSkipSSLValidation bool   `mapstructure:"skipSslValidation"`
	AzureAccountName     string `mapstructure:"azureAccountName"`
	AzureAccountKey      string `mapstructure:"azureAccountKey"`
	UseS3Tags            bool   `mapstructure:"useS3Tags"`
	AWSIAMRole           string `mapstructure:"iamRole"`
}

func (c *ObjectsConfig) UseFileTags() bool {
	return c.CloudName != "azure"
}
