package main

import (
	"github.com/youpipe/go-youPipe/account"
	"net"
	"sync"
)

type Consumer struct {
	sync.RWMutex
	address    account.ID
	aesKey     [32]byte
	publicKey  [32]byte
	privateKey [32]byte
	minerID    string
	minerIP    string
	serPoint   net.Listener
	pipes      map[string]connWaiter
}

type connWaiter struct {
	connId string
	left   net.Conn
	right  net.Conn
}
