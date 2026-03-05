package config

type Config struct {
	Server ServerConfig `koanf:"server"`
	Kafka  KafkaConfig  `koanf:"kafka"`
}

type ServerConfig struct {
	HTTPPort int `koanf:"http_port"`
	GRPCPort int `koanf:"grpc_port"`
}

type KafkaConfig struct {
	Brokers []string `koanf:"brokers"`
	Topic   string   `koanf:"topic"`
	GroupID string   `koanf:"group_id"`
}
