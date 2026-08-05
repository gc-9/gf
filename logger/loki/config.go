package loki

import "time"

// Config configures the Loki client. It lives with the client because it is
// transport configuration rather than application configuration.
type Config struct {
	Enabled bool `yaml:"enabled"`

	URL         string `yaml:"url"`
	TenantID    string `yaml:"tenantID"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	BearerToken string `yaml:"bearerToken"`

	QueueCapacity  int           `yaml:"queueCapacity"`
	BatchSize      int           `yaml:"batchSize"`
	BatchWait      time.Duration `yaml:"batchWait"`
	RequestTimeout time.Duration `yaml:"requestTimeout"`
	MaxRetries     int           `yaml:"maxRetries"`
}
