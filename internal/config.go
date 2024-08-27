package internal

import (
	"github.com/MedvedewEM/didactic-palm-tree/internal/metadb"
	"github.com/alecthomas/units"
	"github.com/sirupsen/logrus"
)

type Config struct {
	App *AppConfig `yaml:"app"`
	Logger *LoggerConfig `yaml:"logger"`
	MetaDB *metadb.MetaDBConfig `yaml:"metadb"`
	Server *ServerConfig `yaml:"server"`
}

func NewDefaultConfig() Config {
	return Config{
		App: NewDefaultAppConfig(),
		Logger: NewDefaultLoggerConfig(),
		MetaDB: metadb.NewDefaultMetaDBConfig(),
		Server: NewDefaultServerConfig(),
	}
}

func MergeConfigSections(cfg Config, newCfg Config) Config {
	oCfg := cfg

	if newCfg.App != nil {
		oCfg.App = newCfg.App
	}

	if newCfg.Logger != nil {
		oCfg.Logger = newCfg.Logger
	}

	if newCfg.MetaDB != nil {
		oCfg.MetaDB = newCfg.MetaDB
	}

	return oCfg
}

type AppConfig struct {
	DistributedStorageServers int `yaml:"distributed_storage_servers"`
}

func NewDefaultAppConfig() *AppConfig {
	return &AppConfig{
		DistributedStorageServers: 2,
	}
}

type LoggerLevel uint32

func (l *LoggerLevel) SetValue(s string) error  {
	switch s {
	case "trace":
		*l = LoggerLevel(logrus.TraceLevel)
	case "debug":
		*l = LoggerLevel(logrus.DebugLevel)
	case "info":
		*l = LoggerLevel(logrus.InfoLevel)
	case "warn":
		*l = LoggerLevel(logrus.WarnLevel)
	case "error":
		*l = LoggerLevel(logrus.ErrorLevel)
	case "fatal":
		*l = LoggerLevel(logrus.FatalLevel)
	case "panic":
		*l = LoggerLevel(logrus.PanicLevel)
	default:
		*l = LoggerLevel(logrus.TraceLevel)
	}
    return nil
}

type LoggerConfig struct {
	Level LoggerLevel `yaml:"level"`
	Output string `yaml:"output"`
}

func NewDefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level: LoggerLevel(logrus.InfoLevel),
		Output: "stdout",
	}
}

type ServerConfig struct {
	Port int `yaml:"port"`
	MaxFileSizeInBytes int64 `yaml:"max_file_size_in_bytes"`
	MaxMemoryUploadFileInBytes int64 `yaml:"max_memory_upload_file_in_bytes"`
}

func NewDefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port: 8080,
		MaxFileSizeInBytes: int64(10 * units.GB),
		MaxMemoryUploadFileInBytes: int64(100 * units.MB),
	}
}
