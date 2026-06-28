package client

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	ServerAddr string
}

func Connect(cfg Config, machineName, user string) error {
	if user == "" {
		fmt.Printf("Connecting to %s\n", machineName)
		fmt.Print("Username: ")
		if _, err := fmt.Scanln(&user); err != nil {
			return err
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	knownHosts := filepath.Join(stateDir(), "known_hosts")

	proxy := proxyCommand(exe, cfg.ServerAddr)
	args := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=" + knownHosts,
		"-o", "HostName=" + machineName,
		"-o", "ProxyCommand=" + proxy,
		"-tt",
		user + "@" + machineName,
	}

	cmd := exec.Command(sshExecutable(), args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh session: %w", err)
	}
	return nil
}

// Proxy bridges stdio to the nossh server for use as ssh ProxyCommand.
func Proxy(cfg Config, machineName string) error {
	conn, err := net.DialTimeout("tcp", cfg.ServerAddr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "CONNECT %s\n", machineName); err != nil {
		return err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "OK") {
		msg := strings.TrimPrefix(line, "ERR ")
		return fmt.Errorf("%s", msg)
	}

	go func() { _, _ = io.Copy(conn, os.Stdin); conn.Close() }()
	_, err = io.Copy(os.Stdout, reader)
	return err
}

func ServerFromEnv(fallback string) string {
	if v := os.Getenv("NOSSH_SERVER"); v != "" {
		return v
	}
	return fallback
}
