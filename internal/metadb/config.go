package metadb

type MetaDBConfig struct {
	Host string `yaml:"host"`
	Port int `yaml:"port"`
	DB string `yaml:"db"`
	User string `yaml:"user"`
	Password string `yaml:"password" env:"METADB_PASSWORD" env-default:""`
}

func NewDefaultMetaDBConfig() *MetaDBConfig {
	return &MetaDBConfig{
		Host: "localhost",
		Port: 5432,
		DB: "default",
		User: "root",
	}
}
