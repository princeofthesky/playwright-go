package config

type Postgres struct {
	Addr string
	User string
	Password string
	Database string
}

type CrawlConfig struct {
	Postgres      Postgres
	SessionKey    string
	CsrfToken     string
	MaxRelated    int
	MaxPageSearch int
}
