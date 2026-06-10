package banner

import (
	"fmt"
	"io"
	"net/url"
	"runtime"
	"strings"
	"time"
)

type Info struct {
	Version string

	UserID int64

	GatewayURL string

	Lookahead   time.Duration
	PreWakeLead time.Duration

	SleepAfterPlanning bool
	DryRun             bool

	LogFile string
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
	fmt.Fprintln(w, `      MORNING PRINTER / LEGACY WAKE PLANNER`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `              .-""""""""-.`)
	fmt.Fprintln(w, `            .'   03:55    '.`)
	fmt.Fprintln(w, `           /   WAKE CLOCK    \`)
	fmt.Fprintln(w, `          |   .----------.    |`)
	fmt.Fprintln(w, `          |   |  pmset   |    |`)
	fmt.Fprintln(w, `          |   | schedule |    |`)
	fmt.Fprintln(w, `           \  '----------'   /`)
	fmt.Fprintln(w, `            '.            .'`)
	fmt.Fprintln(w, `              '-.______.-'`)
	fmt.Fprintln(w, `                  ||||`)
	fmt.Fprintln(w, `             _____||||_____`)
	fmt.Fprintln(w, `            / MAC MINI EDGE \`)
	fmt.Fprintln(w, `            '---------------'`)
	fmt.Fprintln(w)

	printRow(w, "version", valueOrDash(info.Version))
	printRow(w, "node", nodeName())
	printRow(w, "user_id", formatUserID(info.UserID))
	printRow(w, "gateway", safeGatewayLabel(info.GatewayURL))
	printRow(w, "lookahead", formatDuration(info.Lookahead))
	printRow(w, "pre_wake", formatDuration(info.PreWakeLead))
	printRow(w, "sleep_now", formatBool(info.SleepAfterPlanning))
	printRow(w, "dry_run", formatBool(info.DryRun))
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

func formatUserID(userID int64) string {
	if userID <= 0 {
		return "-"
	}

	return fmt.Sprintf("%d", userID)
}

func formatDuration(value time.Duration) string {
	if value <= 0 {
		return "-"
	}

	return value.String()
}

func formatBool(value bool) string {
	if value {
		return "yes"
	}

	return "no"
}

func valueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}

	return value
}
