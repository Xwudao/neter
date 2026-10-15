package core

import (
	"encoding/json"
	"fmt"
)

// WebPublishConfig is consumed by a frontend S3 publishing plugin, not by nr's
// uploader. Credentials are environment variable names only, never secret values.
// Keep JSON names in sync with the frontend plugin's NETER_WEB_PUBLISH_CONFIG contract.
type WebPublishConfig struct {
	// Enabled must serialize false explicitly to override personal .env files.
	Enabled            bool   `yaml:"enabled" json:"enabled"`
	Endpoint           string `yaml:"endpoint" json:"endpoint,omitempty"`
	Region             string `yaml:"region" json:"region,omitempty"`
	Bucket             string `yaml:"bucket" json:"bucket,omitempty"`
	PublicURL          string `yaml:"public_url" json:"public_url,omitempty"`
	Prefix             string `yaml:"prefix" json:"prefix,omitempty"`
	AccessKeyIDEnv     string `yaml:"access_key_id_env" json:"access_key_id_env,omitempty"`
	SecretAccessKeyEnv string `yaml:"secret_access_key_env" json:"secret_access_key_env,omitempty"`
}

type FrontendBuildConfig struct {
	Publish WebPublishConfig `yaml:"publish"`
}

// WebPublishEnv passes non-secret settings to the frontend build process only.
// An explicit false prevents inherited personal .env settings from silently
// changing customer builds. override is non-nil only for an explicit CLI flag.
func (c *NeterConfig) WebPublishEnv(override *bool) ([]string, error) {
	var cfg WebPublishConfig
	if c != nil {
		cfg = c.Build.Frontend.Publish
	}
	if override != nil {
		cfg.Enabled = *override
	}
	if !cfg.Enabled {
		cfg = WebPublishConfig{}
	} else {
		cfg.Endpoint = expandVars(cfg.Endpoint)
		cfg.Region = expandVars(cfg.Region)
		cfg.Bucket = expandVars(cfg.Bucket)
		cfg.PublicURL = expandVars(cfg.PublicURL)
		cfg.Prefix = expandVars(cfg.Prefix)
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode frontend publish config: %w", err)
	}
	return []string{"NETER_WEB_PUBLISH_CONFIG=" + string(data)}, nil
}
