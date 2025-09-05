package bootstrap

import (
	"strings"

	"github.com/anhvanhoa/service-core/bootstrap/config"
	"github.com/anhvanhoa/service-core/domain/grpc_client"
)

type dbCache struct {
	Addr        string `mapstructure:"addr"`
	Db          int    `mapstructure:"db"`
	Password    string `mapstructure:"password"`
	MaxIdle     int    `mapstructure:"max_idle"`
	MaxActive   int    `mapstructure:"max_active"`
	IdleTimeout int    `mapstructure:"idle_timeout"`
	Network     string `mapstructure:"network"`
}

type queue struct {
	Addr        string         `mapstructure:"addr"`
	Db          int            `mapstructure:"db"`
	Password    string         `mapstructure:"password"`
	MaxIdle     int            `mapstructure:"max_idle"`
	MaxActive   int            `mapstructure:"max_active"`
	IdleTimeout int            `mapstructure:"idle_timeout"`
	Network     string         `mapstructure:"network"`
	Concurrency int            `mapstructure:"concurrency"`
	Queues      map[string]int `mapstructure:"queues"`
}

type Env struct {
	NodeEnv string `mapstructure:"node_env"`

	UrlDb string `mapstructure:"url_db"`

	NameService   string `mapstructure:"name_service"`
	PortGrpc      int    `mapstructure:"port_grpc"`
	HostGrpc      string `mapstructure:"host_grpc"`
	IntervalCheck string `mapstructure:"interval_check"`
	TimeoutCheck  string `mapstructure:"timeout_check"`

	DbCache *dbCache `mapstructure:"db_cache"`

	Queue *queue `mapstructure:"queue"`

	GrpcClients []*grpc_client.ConfigGrpc `mapstructure:"grpc_clients"`
}

func NewEnv(env any) {
	setting := config.DefaultSettingsConfig()
	if setting.IsProduction() {
		setting.SetFile("prod.config")
	} else {
		setting.SetFile("dev.config")
	}
	config.NewConfig(setting, env)
}

func (env *Env) IsProduction() bool {
	return strings.ToLower(env.NodeEnv) == "production"
}
