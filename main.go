package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" {
		fmt.Println("usage: ")
		fmt.Println("   upnp [external-ip | eip]                          shows external IP address")
		fmt.Println("   upnp [forward | f] <port> [optional description]  forwards a port")
		fmt.Println("   upnp [clear | c | unforward | uf] <port>          clears (unforwards) a port")
		fmt.Println("   upnp [keep | k] <port> [optional description]     keeps a port forwarded for as long as this program is running")
		fmt.Println("   upnp help                                         shows this message")
		return
	}
	fmt.Println("Finding your router...")
	// connect to router
	routerconn, err := PickRouterClient(context.Background())
	if err != nil {
		fmt.Println("Couldn't find a UPnP router:", err)
		return
	}
	router := CreateRouterClient(routerconn)

	switch os.Args[1] {
	case "external-ip", "eip", "ip":
		// discover external IP
		ip, err := router.GetExternalIPAddress()
		if err != nil {
			fmt.Println("Couldn't discover external IP:", err)
			return
		}
		fmt.Println("Your external IP is:", ip)
	case "forward", "f":
		if len(os.Args) <= 2 {
			fmt.Println("usage:", os.Args[0], "forward <port> [optional description]")
			return
		}
		port, err := strconv.ParseUint(os.Args[2], 10, 16)
		if err != nil {
			fmt.Println("Couldn't parse port to forward:", err)
			return
		}
		desc := "upnp-cli"
		if len(os.Args) > 3 {
			desc = strings.Join(os.Args[3:], " ")
		}
		router.Forward(uint16(port), desc)
		fmt.Println("Forwarded port", port, "successfully!")
		return
	case "clear", "c", "unforward", "uf":
		if len(os.Args) <= 2 {
			fmt.Println("usage:", os.Args[0], "clear <port>")
		}
		port, err := strconv.ParseUint(os.Args[2], 10, 16)
		if err != nil {
			fmt.Println("Couldn't parse port to forward:", err)
			return
		}
		router.Clear(uint16(port))
		fmt.Println("Cleared port", port, "successfully!")
		return

	case "keep", "k":
		if len(os.Args) <= 2 {
			fmt.Println("usage:", os.Args[0], "keep <port> [optional description]")
			return
		}
		port, err := strconv.ParseUint(os.Args[2], 10, 16)
		if err != nil {
			fmt.Println("Couldn't parse port to forward:", err)
			return
		}
		desc := "upnp-cli"
		if len(os.Args) > 3 {
			desc = strings.Join(os.Args[3:], " ")
		}
		err = router.Forward(uint16(port), desc)
		if err != nil {
			fmt.Println("Couldn't forward:", err)
			return
		}
		fmt.Println("Forwarded port", port, "successfully!")

		sigs := make(chan os.Signal, 1)

		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

		fmt.Println("Waiting for a close signal...")
		sig := <-sigs
		fmt.Print("Received signal '", sig, "', clearing port ", port, "...\n")
		router.Clear(uint16(port))
		fmt.Println("Cleared port", port, "successfully!")
		fmt.Println("Exiting...")
		return
	}
}
