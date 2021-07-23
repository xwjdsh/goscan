package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"github.com/xwjdsh/goscan"
)

var iface string

func main() {
	// allow non root user to execute by compare with euid
	if os.Geteuid() != 0 {
		log.Fatal("goscan must run as root.")
	}
	flag.StringVar(&iface, "I", "", "Network interface name")
	flag.Parse()
	// 初始化 网络信息
	goscan.SetupNetInfo(iface)

	ctx, cancel := context.WithCancel(context.Background())
	go goscan.ListenARP(ctx)
	go goscan.ListenMDNS(ctx)
	go goscan.ListenNBNS(ctx)
	go goscan.SendARP()
	go goscan.LocalHost()

	t := time.NewTicker(4 * time.Second)
	for {
		select {
		case <-t.C:
			goscan.PrintData()
			cancel()
			goto END
		case d := <-goscan.Do():
			switch d {
			case goscan.START:
				t.Stop()
			case goscan.END:
				// 接收到新数据，重置2秒的计数器
				t = time.NewTicker(2 * time.Second)
			}
		}
	}
END:
}
