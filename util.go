package main

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/internetgateway2"
	"golang.org/x/sync/errgroup"
)

type RouterClientConn interface {
	GetExternalIPAddress() (
		ExternalIPAddress string,
		err error,
	)
	GetServiceClient() *goupnp.ServiceClient
	AddPortMapping(
		RemoteHost string,
		ExternalPort uint16,
		Protocol string,
		InternalPort uint16,
		InternalClient string,
		Enabled bool,
		PortMappingDescription string,
		LeaseDuration uint32,
	) (err error)
	DeletePortMapping(string, uint16, string) error
	GetSpecificPortMappingEntry(RemoteHost string, ExternalPort uint16, Protocol string) (InternalPort uint16, InternalClient string, Enabled bool, PortMappingDescription string, LeaseDuration uint32, err error)
}

type RouterClient struct {
	RouterClientConn
	localIP string
}

func CreateRouterClient(conn RouterClientConn) RouterClient {
	return RouterClient{
		RouterClientConn: conn,
		localIP:          "",
	}
}

func (r *RouterClient) GetLocalIP() (string, error) {
	if r.localIP != "" {
		return r.localIP, nil
	}
	host, _, _ := net.SplitHostPort(r.GetServiceClient().RootDevice.URLBase.Host)
	devIP := net.ParseIP(host)
	if devIP == nil {
		return "", errors.New("couldn't determine router's internal ip")
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			if x, ok := addr.(*net.IPNet); ok && x.Contains(devIP) {
				return x.IP.String(), nil
			}
		}
	}

	return "", errors.New("couldn't determine internal ip")
}

func (r *RouterClient) Forward(port uint16, desc string) error {
	lip, err := r.GetLocalIP()
	if err != nil {
		return err
	}
	fmt.Println("Forwarding port", port, "on TCP...")
	err = r.AddPortMapping("", port, "TCP", port, lip, true, desc, 0)
	if err != nil {
		return err
	}
	fmt.Println("Forwarding port", port, "on UDP...")
	err = r.AddPortMapping("", port, "UDP", port, lip, true, desc, 0)
	if err != nil {
		return err
	}
	return nil
}

func (r *RouterClient) Clear(port uint16) error {
	fmt.Println("Clearing port", port, "on TCP...")
	err := r.DeletePortMapping("", uint16(port), "TCP")
	if err != nil {
		return err
	}
	fmt.Println("Clearing port", port, "on UDP...")
	err = r.DeletePortMapping("", uint16(port), "UDP")
	if err != nil {
		return err
	}
	return nil
}

func PickRouterClient(ctx context.Context) (RouterClientConn, error) {
	tasks, _ := errgroup.WithContext(ctx)
	// Request each type of client in parallel, and return what is found.
	var ip1Clients []*internetgateway2.WANIPConnection1
	tasks.Go(func() error {
		var err error
		ip1Clients, _, err = internetgateway2.NewWANIPConnection1Clients()
		return err
	})
	var ip2Clients []*internetgateway2.WANIPConnection2
	tasks.Go(func() error {
		var err error
		ip2Clients, _, err = internetgateway2.NewWANIPConnection2Clients()
		return err
	})
	var ppp1Clients []*internetgateway2.WANPPPConnection1
	tasks.Go(func() error {
		var err error
		ppp1Clients, _, err = internetgateway2.NewWANPPPConnection1Clients()
		return err
	})

	if err := tasks.Wait(); err != nil {
		return nil, err
	}

	// Trivial handling for where we find exactly one device to talk to, you
	// might want to provide more flexible handling than this if multiple
	// devices are found.
	switch {
	case len(ip2Clients) == 1:
		return ip2Clients[0], nil
	case len(ip1Clients) == 1:
		return ip1Clients[0], nil
	case len(ppp1Clients) == 1:
		return ppp1Clients[0], nil
	default:
		return nil, errors.New("multiple or no services found")
	}
}
