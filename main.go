package main

import (
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/go-logger/logger"
	hfw "github.com/hsyan2008/hfw2"
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
		tmpServers[v.Group] = v.SSHConfig
	}
	for _, v := range config.Inner {
		l, err := ssh.NewLocalForward(tmpServers[v.Group], v.ForwardIni)
		defer l.Close()
		if err != nil {
			logger.Error(v, err)
			return
		}
	}

	_ = hfw.Run()
}

type tomlConfig struct {
	Title  string
	Keep   time.Duration
	Server []server
	Inner  []inner
}

type server struct {
	Group string `toml:"group"`
	ssh.SSHConfig
}

type inner struct {
	Group string `toml:"group"`
	ssh.ForwardIni
}
