package main

import (
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/go-logger/logger"
	"github.com/hsyan2008/hfw2/ssh"
)

var config tomlConfig

func init() {
	if _, err := toml.DecodeFile("main.toml", &config); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func main() {
	var tmpServers = map[string]ssh.SSHConfig{}
	for _, v := range config.Server {
		sshConfig := ssh.SSHConfig{
			Addr: v.Addr,
			User: v.User,
			Auth: v.Auth,
		}
		tmpServers[v.Group] = sshConfig
	}
	for _, v := range config.Inner {
		l, err := ssh.NewLocal(tmpServers[v.Group], v.Bind, v.Addr)
		if err != nil {
			logger.Error(v, err)
			return
		}
		go func() {
			_ = l.Do()
		}()
	}

	select {}
}

type tomlConfig struct {
	Title  string
	Keep   time.Duration
	Server []server
	Inner  []inner
}

type server struct {
	Group string `toml:"group"`
	Addr  string `toml:"addr"`
	User  string `toml:"user"`
	Auth  string `toml:"auth"`
}

type inner struct {
	Group string `toml:"group"`
	Addr  string `toml:"addr"`
	Bind  string `toml:"bind"`
}
