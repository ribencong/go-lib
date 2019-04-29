package androidDelegate

import "syscall"

type VpnService interface {
	Protect(fd int) bool
}

type LogService interface {
	InsertProxyLog(log string, level int)
}

type PacketFlow interface {
	WritePacket(packet []byte)
}

func InputPacket(data []byte) {
}

func SetNonblock(fd int, nonblocking bool) bool {
	err := syscall.SetNonblock(fd, nonblocking)
	if err != nil {
		return false
	}
	return true
}

func SetLocalDNS(dns string) {
}

func StartTun(packetFlow PacketFlow, vpnService VpnService, logService LogService) {
}

func StopTun() {
}
