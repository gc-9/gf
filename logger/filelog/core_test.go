package filelog

import "testing"

func TestConfigDefaults(t *testing.T) {
	config := (Config{Filename: "app.log"}).withDefaults()
	if config.Filename != "app.log" || config.MaxSize != DefaultMaxSize || config.MaxBackups != DefaultMaxBackups || config.MaxAge != DefaultMaxAge {
		t.Fatalf("defaults = %+v", config)
	}
}
