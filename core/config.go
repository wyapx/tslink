package core

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/BurntSushi/toml"
)

type ForwardRule struct {
	Protocol      string `toml:"protocol"`
	TailscalePort int    `toml:"tailscale_port"`
	LocalAddr     string `toml:"local_addr"`
}

type ConnectRule struct {
	Protocol  string `toml:"protocol"`
	LocalPort int    `toml:"local_port"`
	LocalAddr string `toml:"local_addr"`
	DstAddr   string `toml:"dst_addr"`
	LanEnable *bool  `toml:"lan_enable"`
	LanMotd   string `toml:"lan_motd"`
}

type Core struct {
	AuthKey      string `toml:"auth_key"`
	ControlURL   string `toml:"control_url"`
	Hostname     string `toml:"hostname"`
	Ephemeral    bool   `toml:"ephemeral"`
	AcceptRoutes bool   `toml:"accept_routes"`
}

func (r ConnectRule) LANEnabled() bool {
	if r.LanEnable != nil {
		return *r.LanEnable
	}
	return r.Protocol == "minecraft"
}

func (r ConnectRule) LANMotdOr(def string) string {
	if r.LanMotd != "" {
		return r.LanMotd
	}
	return def
}

type Config struct {
	Core    Core                     `toml:"core"`
	Forward map[string][]ForwardRule `toml:"forward"`
	Connect map[string][]ConnectRule `toml:"connect"`
}

func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Core: Core{
			Hostname:     "",
			Ephemeral:    true,
			AcceptRoutes: true,
		},
		Forward: make(map[string][]ForwardRule),
		Connect: make(map[string][]ConnectRule),
	}
	if isRemoteConfigPath(path) {
		return loadRemoteConfig(path, cfg)
	}

	if _, err := os.Stat(path); err != nil {
		// write a basic config
		buf := new(bytes.Buffer)
		err = toml.NewEncoder(buf).Encode(cfg)
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(path, buf.Bytes(), 0644)
	}
	_, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Core.Hostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		cfg.Core.Hostname = hostname
	}

	return cfg, nil
}

func isRemoteConfigPath(path string) bool {
	u, err := url.Parse(path)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func loadRemoteConfig(path string, cfg *Config) (*Config, error) {
	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download config: unexpected status %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, err
	}

	if cfg.Core.Hostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		cfg.Core.Hostname = hostname
	}

	return cfg, nil
}
