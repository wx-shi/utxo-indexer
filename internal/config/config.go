package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

// Config holds the configuration settings for the application.
type Config struct {
	Server   *ServerConfig     `yaml:"server"`
	LogLevel string            `yaml:"log_level"`
	BadgerDB *BadgerDBConfig   `yaml:"badger_db"`
	DB       *DBConfig         `yaml:"db"`
	RPC      *BitcoinRPCConfig `yaml:"rpc"`
	Indexer  *IndexerConfig    `yaml:"indexer"`
}

// ServerConfig holds the configuration settings for the HTTP server.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// BadgerDBConfig holds the configuration settings for BadgerDB.
type BadgerDBConfig struct {
	Directory string `yaml:"directory"`
}

type DBConfig struct {
	Name   string `yaml:"name"`
	Dir    string `yaml:"dir"`
	DBType string `yaml:"db_type"`
}

// BitcoinRPCConfig holds the configuration settings for Bitcoin JSON-RPC.
type BitcoinRPCConfig struct {
	URL      string `yaml:"url"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type IndexerConfig struct {
	BatchSize    int `yaml:"batch_size"`     //阈值 累计达到该值进行存储 len(vin)+len(vout)
	BlockChanBuf int `yaml:"block_chan_buf"` //在存储过程中还可以查询该缓冲区大小个块
}

// LoadConfig reads and parses the configuration file.
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
