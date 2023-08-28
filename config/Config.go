package config

type Postgres struct {
	Addr     string
	User     string
	Password string
	Database string
}

type Elasticsearch struct {
	Addr     string
	User     string
	Password string
}

type CrawlConfig struct {
	Postgres      Postgres
	SessionKey    string
	CsrfToken     string
	MaxThread     int
	MaxPage       int
	MaxRetry      int
	StartRegion   int
	ProxyTestUrl  string
	NginxProxy    string
	Elasticsearch Elasticsearch
}
