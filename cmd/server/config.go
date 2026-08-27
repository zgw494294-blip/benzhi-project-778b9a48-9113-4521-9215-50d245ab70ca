package main

import (
	"errors"
	"net"
	"os"
	"strings"
)

type config struct {
	addr, dbPath string
	selfcheck    bool
}

func defaultAddress() string {
	if p := os.Getenv("PORT"); p != "" {
		if _, e := net.LookupPort("tcp", p); e == nil {
			return net.JoinHostPort("127.0.0.1", p)
		}
	}
	return "127.0.0.1:19091"
}
func validateAddress(addr string) error {
	if strings.TrimSpace(addr) == "" {
		return errors.New("-addr 不能为空")
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return errors.New("-addr 必须为 host:port")
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return errors.New("拒绝绑定全网地址，请明确使用回环或指定接口")
	}
	return nil
}
