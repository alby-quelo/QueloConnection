package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nossh/nossh/internal/admin"
	"github.com/nossh/nossh/internal/codes"
	"github.com/nossh/nossh/internal/client"
)

func main() {
	serverAddr := flag.String("server", client.ServerFromEnv("127.0.0.1:7000"), "bridge client address")
	adminURL := flag.String("admin-url", os.Getenv("NOSSH_ADMIN_URL"), "admin API base URL")
	adminToken := flag.String("admin-token", os.Getenv("NOSSH_ADMIN_TOKEN"), "admin API token")
	flag.Parse()

	if *adminURL == "" {
		*adminURL = "http://127.0.0.1:8081"
	}

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	switch args[0] {
	case "connect":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: nossh connect <machine-name> [user]")
			os.Exit(1)
		}
		user := ""
		if len(args) > 2 {
			user = args[2]
		}
		cfg := client.Config{ServerAddr: *serverAddr}
		if err := client.Connect(cfg, args[1], user); err != nil {
			fmt.Fprintf(os.Stderr, "connect: %v\n", err)
			os.Exit(1)
		}

	case "proxy":
		if len(args) < 2 {
			os.Exit(1)
		}
		cfg := client.Config{ServerAddr: *serverAddr}
		if err := client.Proxy(cfg, args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "proxy: %v\n", err)
			os.Exit(1)
		}

	case "list":
		c := adminClient(*adminURL, *adminToken)
		agents, err := c.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "list: %v\n", err)
			os.Exit(1)
		}
		admin.PrintList(agents)

	case "name":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: nossh name <CODE> <machine-name>")
			os.Exit(1)
		}
		c := adminClient(*adminURL, *adminToken)
		rec, err := c.AssignName(codes.Normalize(args[1]), args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "name: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("OK: %s is now reachable as %q\n", rec.Code, rec.Name)

	case "rename":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: nossh rename <old-name> <new-name>")
			os.Exit(1)
		}
		c := adminClient(*adminURL, *adminToken)
		rec, err := c.Rename(args[1], args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "rename: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("OK: renamed to %q\n", rec.Name)

	case "revoke":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: nossh revoke <machine-name>")
			os.Exit(1)
		}
		c := adminClient(*adminURL, *adminToken)
		if err := c.Revoke(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "revoke: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("OK: revoked")

	case "delete":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: nossh delete <CODE>")
			os.Exit(1)
		}
		c := adminClient(*adminURL, *adminToken)
		if err := c.Delete(codes.Normalize(args[1])); err != nil {
			fmt.Fprintf(os.Stderr, "delete: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("OK: deleted")

	default:
		usage()
		os.Exit(1)
	}
}

func adminClient(url, token string) admin.Client {
	return admin.Client{BaseURL: strings.TrimRight(url, "/"), Token: token}
}

func usage() {
	fmt.Fprintf(os.Stderr, `nossh — SSH via bridge without knowing the remote IP

Client:
  nossh connect <machine-name> [user]
  nossh proxy <machine-name>          (used internally by ssh ProxyCommand)

Admin (run on bridge server):
  nossh list
  nossh name <CODE> <machine-name>
  nossh rename <old-name> <new-name>
  nossh revoke <machine-name>
  nossh delete <CODE>

Environment:
  NOSSH_SERVER       bridge address (default 127.0.0.1:7000)
  NOSSH_ADMIN_URL    admin API (default http://127.0.0.1:8081)
  NOSSH_ADMIN_TOKEN  admin bearer token
`)
}
