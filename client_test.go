package main

import (
	"github.com/youpipe/go-youPipe/service/client"
	"testing"
)

func TestClient(t *testing.T) {
	conf := &client.Config{
		Addr:        "YPAGC4RKAh2gUUQQ1SeEMbWbtzSoHgkPAiWxyhy4EDz7iy",
		Cipher:      "2n6mmNWTLn6UN6CFtRX9pEdxR2VFc3MwvcytAniQmRGaLrsbYmxaAE6jLakBPYKBUihfT578uT9ctbF2P5Uy21j9BgPVuoXyQZGC6x6ir58QgT",
		LocalServer: ":1080",
		Services:    []string{"YPBzFaBFv8ZjkPQxtozNQe1c9CvrGXYg4tytuWjo9jiaZx@192.168.1.108"},
	}

	cli, err := client.NewClient(conf, "12345678")
	if err != nil {
		t.Fatalf("create new client failed:%v", err)
	}

	if err := cli.Running(); err != nil {
		t.Fatalf(err.Error())
	}
}
