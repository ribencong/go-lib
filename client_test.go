package main

import (
	"github.com/ribencong/go-lib/client"
	"testing"
)

func TestClient(t *testing.T) {
	conf := &client.Config{
		Addr:        "YPAGC4RKAh2gUUQQ1SeEMbWbtzSoHgkPAiWxyhy4EDz7iy",
		Cipher:      "2n6mmNWTLn6UN6CFtRX9pEdxR2VFc3MwvcytAniQmRGaLrsbYmxaAE6jLakBPYKBUihfT578uT9ctbF2P5Uy21j9BgPVuoXyQZGC6x6ir58QgT",
		LocalServer: ":1080",
	}

	cli, err := client.NewClient(conf, "12345678")
	if err != nil {
		t.Fatalf("create new client failed:%v", err)
	}

	if err := cli.Running(); err != nil {
		t.Fatalf(err.Error())
	}
}
