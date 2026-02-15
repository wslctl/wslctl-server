package wslctl_server

import "time"

type Distribution struct {
	Boot     Boot     `yaml:"boot"`
	Metadata Metadata `yaml:"metadata"`
	OS       OS       `yaml:"os"`
	User     User     `yaml:"user"`
	State    string   `yaml:"state"`
	System   System   `yaml:"system"`
	Config   Config   `yaml:"config"`
}

type Boot struct {
	Systemd Systemd `yaml:"systemd"`
}

type Systemd struct {
	Enabled bool `yaml:"enabled"`
}

type Metadata struct {
	Name           string    `yaml:"name"`
	Base           string    `yaml:"base"`
	KernelVersion  string    `yaml:"kernel-version"`
	Version        int       `yaml:"version"`
	Architecture   string    `yaml:"architecture"`
	LastUsed       time.Time `yaml:"last-used"`
	InstalledOn    time.Time `yaml:"installed-on"`
	WindowsVersion string    `yaml:"windows-version"`
}

type OS struct {
	ID            string    `yaml:"id"`
	Name          string    `yaml:"name"`
	Version       OSVersion `yaml:"version"`
	SupportStatus string    `yaml:"support-status"`
}

type OSVersion struct {
	ID       string `yaml:"id"`
	Codename string `yaml:"codename"`
}

type User struct {
	Current string `yaml:"current"`
	Default string `yaml:"default"`
}

type System struct {
	Disk    Disk    `yaml:"disk"`
	CPU     CPU     `yaml:"cpu"`
	Memory  Memory  `yaml:"memory"`
	Network Network `yaml:"network"`
	Uptime  Uptime  `yaml:"uptime"`
}

type Disk struct {
	Size   ValueUnit `yaml:"size"`
	Usage  float64   `yaml:"usage"`
	Used   ValueUnit `yaml:"used"`
	Free   ValueUnit `yaml:"free"`
	Sparse bool      `yaml:"sparse"`
}

type ValueUnit struct {
	Value int    `yaml:"value"`
	Unit  string `yaml:"unit"`
}

type CPU struct {
	Cores CPUCores `yaml:"cores"`
	Usage float64  `yaml:"usage"`
}

type CPUCores struct {
	Count int            `yaml:"count"`
	Usage []CPUCoreUsage `yaml:"usage"`
}

type CPUCoreUsage struct {
	Core  int     `yaml:"core"`
	Usage float64 `yaml:"usage"`
}

type Memory struct {
	Size      ValueUnit `yaml:"size"`
	Usage     float64   `yaml:"usage"`
	Used      ValueUnit `yaml:"used"`
	Free      ValueUnit `yaml:"free"`
	Shared    ValueUnit `yaml:"shared"`
	Cached    ValueUnit `yaml:"cached"`
	Available ValueUnit `yaml:"available"`
}

type Network struct {
	Interfaces  []NetworkInterface `yaml:"interfaces"`
	Routes      []NetworkRoute     `yaml:"routes"`
	Nameservers []Nameserver       `yaml:"nameservers"`
}

type NetworkInterface struct {
	Name      string           `yaml:"name"`
	Type      string           `yaml:"type"`
	Speed     ValueUnit        `yaml:"speed"`
	Addresses []NetworkAddress `yaml:"addresses"`
}

type NetworkAddress struct {
	Mac  string `yaml:"mac"`
	Ipv4 string `yaml:"ipv4"`
	Ipv6 string `yaml:"ipv6"`
}

type NetworkRoute struct {
	Gateway NetworkGateway `yaml:"gateway"`
}

type NetworkGateway struct {
	Address   string           `yaml:"address"`
	Interface NetworkInterface `yaml:"interface"`
}

type Nameserver struct {
	Address string `yaml:"address"`
}

type Uptime struct {
	CurrentTime  string       `yaml:"current-time"`
	System       string       `yaml:"system"`
	Users        int          `yaml:"users"`
	LoadAverages LoadAverages `yaml:"load-averages"`
}

type LoadAverages struct {
	One     float64 `yaml:"one"`
	Five    float64 `yaml:"five"`
	Fifteen float64 `yaml:"fifteen"`
}

type Config struct {
	Limits            Limits `yaml:"limits"`
	NetworkingMode    string `yaml:"networking-mode"`
	IntegratedConsole bool   `yaml:"integrated-console"`
}

type Limits struct {
	Memory ValueUnit `yaml:"memory"`
	CPU    int       `yaml:"cpu"`
}

type DistributionsList struct {
	Distributions []Distribution `yaml:"distributions"`
}
