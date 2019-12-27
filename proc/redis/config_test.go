package redis

import (
	"time"

	"github.com/samaritan-proxy/samaritan/pb/config/protocol"
	"github.com/samaritan-proxy/samaritan/pb/config/service"
	"github.com/samaritan-proxy/samaritan/utils"
)

func makeDefaultConfig() *config {
	redisOption := &protocol.RedisOption{}
	raw := &service.Config{
		ConnectTimeout:  utils.DurationPtr(time.Second),
		Protocol:        protocol.Redis,
		ProtocolOptions: &service.Config_RedisOption{RedisOption: redisOption},
	}
	return newConfig(raw)
}
