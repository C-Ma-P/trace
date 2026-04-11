package mdns

import (
	"net"
	"time"
)

const (
	mcastIPv4        = "224.0.0.251"
	mcastPort        = 5353
	hostTTL          = uint32(120)
	legacyMaxTTL     = uint32(10)
	announceInterval = 25 * time.Second
	checkDelay       = 750 * time.Millisecond
	checkTimeout     = 3 * time.Second
)

var mcastGroupAddr = &net.UDPAddr{IP: net.ParseIP(mcastIPv4), Port: mcastPort}

type Config struct {
	Hostname string
	IPv4     string
	Hooks    Hooks
}

type Hooks struct {
	Info    func(kind, msg string, meta map[string]any)
	Success func(kind, msg string, meta map[string]any)
	Warn    func(kind, msg string, meta map[string]any)
	Error   func(kind, msg string, meta map[string]any)
}

func (h Hooks) info(kind, msg string, meta map[string]any) {
	if h.Info != nil {
		h.Info(kind, msg, meta)
	}
}

func (h Hooks) success(kind, msg string, meta map[string]any) {
	if h.Success != nil {
		h.Success(kind, msg, meta)
	}
}

func (h Hooks) warn(kind, msg string, meta map[string]any) {
	if h.Warn != nil {
		h.Warn(kind, msg, meta)
	}
}

func (h Hooks) fail(kind, msg string, meta map[string]any) {
	if h.Error != nil {
		h.Error(kind, msg, meta)
	}
}
