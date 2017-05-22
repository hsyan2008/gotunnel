package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/BurntSushi/toml"
)

var config tomlConfig
var servers = make(map[string]*ssh.Client)
var lock = new(sync.Mutex)

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
	c := connectServer(s)

	lister, err := net.Listen("tcp", i.Bind)
	if err != nil {
		log.Println("bind error:", err)
	}
	defer func() {
		_ = lister.Close()
	}()

	for {
		localConn, err := lister.Accept()
		if err != nil {
			log.Println("lister error:", err)
		}

		go hand(c, localConn, i)
	}
}

func hand(c *ssh.Client, localConn net.Conn, i inner) {
	sshConn, err := c.Dial("tcp", i.Addr)
	if err != nil {
		log.Println("dial remote error:", err)
	}

	go copyNet(localConn, sshConn)
	go copyNet(sshConn, localConn)
}

//考虑用select来改进
func copyNet(src, des net.Conn) {
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
		log.Println("io copy error:", err)
	}
}

func connectServer(s server) (c *ssh.Client) {
	lock.Lock()
	defer lock.Unlock()
	c, ok := servers[s.Group]
	if ok {
		return c
	}

	config := &ssh.ClientConfig{
		User: s.User,
		Auth: []ssh.AuthMethod{
			getAuth(s.Auth),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	c, err := ssh.Dial("tcp", s.Addr, config)

	if err != nil {
		log.Println("connect server error:", err)
	}

	servers[s.Group] = c

	return c
}

func getAuth(auth string) ssh.AuthMethod {
	//是文件
	var key []byte

	if _, err := os.Stat(auth); err == nil {
		key, _ = ioutil.ReadFile(auth)
	} else {
		log.Println("read key file error:", err)
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
		log.Println("parse private key error:", err)
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
