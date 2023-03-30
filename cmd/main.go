package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/ftl/clusterix"
)

var rootFlags = struct {
	hostAddress string
	reconnect   bool
	username    string
	password    string
	trace       bool
}{}

var rootCmd = &cobra.Command{
	Use:   "clusterix",
	Short: "A simple tool to access DX clusters.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&rootFlags.hostAddress, "host", "", "connect to this DX cluster host")
	rootCmd.PersistentFlags().BoolVar(&rootFlags.reconnect, "reconnect", false, "try to reconnect if the DX cluster connection failed")
	rootCmd.PersistentFlags().StringVar(&rootFlags.username, "username", "", "the username, usually your callsign")
	rootCmd.PersistentFlags().StringVar(&rootFlags.password, "password", "", "the password")
	rootCmd.PersistentFlags().BoolVar(&rootFlags.trace, "trace", false, "trace the communication on the console")
}

func main() {
	Execute()
}

func runWithClient(f func(context.Context, *clusterix.Client, *cobra.Command, []string)) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		host, err := parseHostArg(rootFlags.hostAddress)
		if err != nil {
			log.Fatalf("invalid host address: %v", err)
		}
		if host.Port == 0 {
			host.Port = clusterix.DefaultPort
			log.Printf("using the default port %d", host.Port)
		}

		ctx, cancel := context.WithCancel(context.Background())
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		go handleCancelation(signals, cancel)

		var c *clusterix.Client
		if rootFlags.reconnect {
			c = clusterix.KeepOpen(host, rootFlags.username, rootFlags.password, 30*time.Second, rootFlags.trace)
		} else {
			c, err = clusterix.Open(host, rootFlags.username, rootFlags.password, rootFlags.trace)
		}
		if err != nil {
			log.Fatalf("cannot conntect to %s: %v", host.String(), err)
		}
		defer c.Disconnect()
		if !rootFlags.reconnect {
			c.WhenDisconnected(cancel)
		}

		f(ctx, c, cmd, args)
	}
}

func handleCancelation(signals <-chan os.Signal, cancel context.CancelFunc) {
	count := 0
	for _ = range signals {
		count++
		if count == 1 {
			cancel()
		} else {
			log.Fatal("hard shutdown")
		}
	}
}

func parseHostArg(arg string) (*net.TCPAddr, error) {
	host, port := splitHostPort(arg)
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = strconv.Itoa(clusterix.DefaultPort)
	}

	return net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%s", host, port))
}

func splitHostPort(hostport string) (host, port string) {
	host = hostport

	colon := strings.LastIndexByte(host, ':')
	if colon != -1 && validOptionalPort(host[colon:]) {
		host, port = host[:colon], host[colon+1:]
	}

	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}

	return
}

func validOptionalPort(port string) bool {
	if port == "" {
		return true
	}
	if port[0] != ':' {
		return false
	}
	for _, b := range port[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}
