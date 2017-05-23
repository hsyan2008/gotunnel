package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/BurntSushi/toml"
	"github.com/hsyan2008/go-logger/logger"
)

var config tomlConfig

func init() {
	if _, err := toml.DecodeFile("main.toml", &config); err != nil {
		fmt.Println(err)
		panic("load config error")
	}
}

func main() {
	var servers = map[string]server{}
	for _, v := range config.Server {
		servers[v.Group] = v
	}
	for _, v := range config.Inner {
		go connect(servers[v.Group], v)
	}

	for {
		time.Sleep(10 * time.Second)
	}
}

func connect(s server, i inner) {

	lister, err := net.Listen("tcp", i.Bind)
	if err != nil {
		logger.Warn("bind error:", err)
		return
	}
	defer func() {
		_ = lister.Close()
	}()

	for {
		localConn, err := lister.Accept()
		if err != nil {
			logger.Warn("lister error:", err)
		}

		go hand(localConn, s, i)
	}
}

func hand(localConn net.Conn, s server, i inner) {
	c := connectServer(s)
	defer func() {
		_ = c.Close()
	}()

	sshConn, err := c.Dial("tcp", i.Addr)
	if err != nil {
		logger.Warn("dial remote error:", err)
		return
	}

	var ch = make(chan bool, 0)
	go copyNet(ch, localConn, sshConn)
	go copyNet(ch, sshConn, localConn)
	<-ch
	<-ch
}

//考虑用select来改进
func copyNet(ch chan bool, src, des net.Conn) {
	defer func() {
		//当ssh退出的时候，只是退出了远程连接，而本地连接未中断
		//当在linux下还好，在window的xshell下，会被挂起
		//所以双向拷贝，只有一个退出，所以直接两个都关闭，确保都退出
		_ = src.Close()
		_ = des.Close()
	}()
	_, err := io.Copy(des, src)
	if err != nil {
		//因为双向拷贝，有一个是正常退出，另一个是被迫关闭的，所以出现一个错误是正常的
		logger.Warn("io copy error:", err)
	}
	ch <- true
}

func connectServer(s server) (c *ssh.Client) {

	config := &ssh.ClientConfig{
		User: s.User,
		Auth: []ssh.AuthMethod{
			getAuth(s.Auth),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	c, err := ssh.Dial("tcp", s.Addr, config)

	if err != nil {
		logger.Warn("connect server error:", err)
		return
	}

	return c
}

func getAuth(auth string) ssh.AuthMethod {
	//是文件
	var key []byte

	if _, err := os.Stat(auth); err == nil {
		key, _ = ioutil.ReadFile(auth)
	} else {
		logger.Warn("read key file error:", err)
		return nil
	}

	//密码
	if len(key) == 0 {
		if len(auth) < 50 {
			return ssh.Password(auth)
		} else {
			key = []byte(auth)
		}
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		logger.Warn("parse private key error:", err)
	}

	return ssh.PublicKeys(signer)
}

type tomlConfig struct {
	Title  string
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
