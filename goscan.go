package goscan

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	manuf "github.com/timest/gomanuf"
)

var log = logrus.New()

// ipNet 存放 IP地址和子网掩码
var ipNet *net.IPNet

// 本机的mac地址，发以太网包需要用到
var localHaddr net.HardwareAddr

// 存放最终的数据，key[string] 存放的是IP地址
var data map[string]Info = make(map[string]Info)

var do chan string = make(chan string)

var iface string

const (
	// 3秒的计时器
	START = "start"
	END   = "end"
)

type Info struct {
	// IP地址
	Mac net.HardwareAddr
	// 主机名
	Hostname string
	// 厂商信息
	Manuf string
}

// 格式化输出结果
// xxx.xxx.xxx.xxx  xx:xx:xx:xx:xx:xx  hostname  manuf
// xxx.xxx.xxx.xxx  xx:xx:xx:xx:xx:xx  hostname  manuf
func PrintData() {
	var keys IPSlice
	for k := range data {
		keys = append(keys, ParseIPString(k))
	}
	sort.Sort(keys)
	for _, k := range keys {
		d := data[k.String()]
		mac := ""
		if d.Mac != nil {
			mac = d.Mac.String()
		}
		fmt.Printf("%-15s %-17s %-30s %-10s\n", k.String(), mac, d.Hostname, d.Manuf)
	}
}

// 将抓到的数据集加入到data中，同时重置计时器
func pushData(ip string, mac net.HardwareAddr, hostname, manuf string) {
	// 停止计时器
	do <- START
	var mu sync.RWMutex
	mu.RLock()
	defer func() {
		// 重置计时器
		do <- END
		mu.RUnlock()
	}()
	if _, ok := data[ip]; !ok {
		data[ip] = Info{Mac: mac, Hostname: hostname, Manuf: manuf}
		return
	}
	info := data[ip]
	if len(hostname) > 0 && len(info.Hostname) == 0 {
		info.Hostname = hostname
	}
	if len(manuf) > 0 && len(info.Manuf) == 0 {
		info.Manuf = manuf
	}
	if mac != nil {
		info.Mac = mac
	}
	data[ip] = info
}

func SetupNetInfo(f string) {
	iface = f
	var ifs []net.Interface
	var err error
	if f == "" {
		ifs, err = net.Interfaces()
	} else {
		// 已经选择iface
		var it *net.Interface
		it, err = net.InterfaceByName(f)
		if err == nil {
			ifs = append(ifs, *it)
		}
	}
	if err != nil {
		log.Fatal("无法获取本地网络信息:", err)
	}
	for _, it := range ifs {
		addr, _ := it.Addrs()
		for _, a := range addr {
			if ip, ok := a.(*net.IPNet); ok && !ip.IP.IsLoopback() {
				if ip.IP.To4() != nil {
					ipNet = ip
					localHaddr = it.HardwareAddr
					iface = it.Name
					goto END
				}
			}
		}
	}
END:
	if ipNet == nil || len(localHaddr) == 0 {
		log.Fatal("无法获取本地网络信息")
	}
}

func LocalHost() {
	host, _ := os.Hostname()
	data[ipNet.IP.String()] = Info{Mac: localHaddr, Hostname: strings.TrimSuffix(host, ".local"), Manuf: manuf.Search(localHaddr.String())}
}

func SendARP() {
	// ips 是内网IP地址集合
	ips := Table(ipNet)
	for _, ip := range ips {
		go SendArpPackage(ip)
	}
}

func Do() <-chan string {
	return do
}
