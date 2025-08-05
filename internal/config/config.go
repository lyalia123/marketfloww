package config

type Config struct {
	Mode      string      `yaml:"mode"`
	Postgres  PostgresCfg `yaml:"postgres"`
	Redis     RedisCfg    `yaml:"redis"`
	Exchanges []string    `yaml:"exchanges"`
}

type PostgresCfg struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type RedisCfg struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
}
