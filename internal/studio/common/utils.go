package common

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"time"
)

// FindAvailablePort finds an available port starting from startPort
func FindAvailablePort(startPort int) int {
	for port := startPort; port < startPort+100; port++ {
		if isPortAvailable(port) {
			return port
		}
	}
	return startPort
}

func isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return false
	}
	ln.Close()
	
	time.Sleep(10 * time.Millisecond)
	
	conn, err := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
	if err == nil {
		conn.Close()
		return false 
	}
	
	return true
}

// OpenBrowser opens the default browser with the given URL
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// QuoteIdentifier quotes a SQL identifier
func QuoteIdentifier(name string) string {
	return fmt.Sprintf("\"%s\"", name)
}
