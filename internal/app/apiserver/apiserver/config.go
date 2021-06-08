package apiserver

type Config struct {
	Port      string `toml:"port"`
	Url       string `toml:"database_url"`
	Level     string `toml:"log_level"`
	Secretkey string `toml:"secret_key"`
}
