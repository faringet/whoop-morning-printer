package banner

import (
	"fmt"
	"io"
	"net/url"
	"runtime"
	"strings"
)

type Info struct {
	Version string

	Mode string

	OutputMode  string
	PrinterName string
	CPI         int
	LPI         int

	GatewayURL string
	LogFile    string
}

func Print(w io.Writer, info Info) {
	if w == nil {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, ` __        ___   _  ___   ___  ____  `)
	fmt.Fprintln(w, ` \ \      / / | | |/ _ \ / _ \|  _ \ `)
	fmt.Fprintln(w, `  \ \ /\ / /| |_| | | | | | | | |_) |`)
	fmt.Fprintln(w, `   \ V  V / |  _  | |_| | |_| |  __/ `)
	fmt.Fprintln(w, `    \_/\_/  |_| |_|\___/ \___/|_|    `)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `      MORNING PRINTER / LEGACY PRINTER AGENT`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `          .----------------------.`)
	fmt.Fprintln(w, `       .-'                        '-.`)
	fmt.Fprintln(w, `     .'        __________________       '.`)
	fmt.Fprintln(w, `    /        .'   STAR SP712      '.      \`)
	fmt.Fprintln(w, `   /________/   9-PIN IMPACT       \_______\`)
	fmt.Fprintln(w, `   |        |                      |       |`)
	fmt.Fprintln(w, `   |        |                      |       |`)
	fmt.Fprintln(w, `   |        |                      |       |`)
	fmt.Fprintln(w, `   |        |                      |       |`)
	fmt.Fprintln(w, `   |        |______________________|       |`)
	fmt.Fprintln(w, `   \______________________________________/`)
	fmt.Fprintln(w, `       \__\                          /__/`)
	fmt.Fprintln(w)

	printRow(w, "version", valueOrDash(info.Version))
	printRow(w, "node", nodeName())
	printRow(w, "mode", valueOrDash(info.Mode))
	printRow(w, "output", valueOrDash(info.OutputMode))
	printRow(w, "printer", valueOrDash(info.PrinterName))
	printRow(w, "lp", lpMode(info.CPI, info.LPI))
	printRow(w, "gateway", safeGatewayLabel(info.GatewayURL))
	printRow(w, "logs", valueOrDash(info.LogFile))

	fmt.Fprintln(w)
}

func printRow(w io.Writer, key string, value string) {
	fmt.Fprintf(w, "  %-11s %s\n", key, value)
}

func nodeName() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac edge"
	case "windows":
		return "windows dev node"
	case "linux":
		return "linux node"
	default:
		return runtime.GOOS + " node"
	}
}

func lpMode(cpi int, lpi int) string {
	if cpi <= 0 && lpi <= 0 {
		return "-"
	}

	if cpi <= 0 {
		return fmt.Sprintf("lpi=%d", lpi)
	}

	if lpi <= 0 {
		return fmt.Sprintf("cpi=%d", cpi)
	}

	return fmt.Sprintf("cpi=%d lpi=%d", cpi, lpi)
}

func safeGatewayLabel(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "-"
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return sanitizeRawURL(rawURL)
	}

	host := strings.TrimSpace(u.Host)
	if host == "" {
		return sanitizeRawURL(rawURL)
	}

	path := strings.TrimSpace(u.Path)
	if path == "" || path == "/" {
		return host
	}

	return host + path
}

func sanitizeRawURL(rawURL string) string {
	if i := strings.Index(rawURL, "?"); i >= 0 {
		rawURL = rawURL[:i]
	}

	if i := strings.LastIndex(rawURL, "@"); i >= 0 {
		rawURL = rawURL[i+1:]
	}

	return valueOrDash(rawURL)
}

func valueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}

	return value
}
