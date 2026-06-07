package lifecycle

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

type ShutdownBannerInfo struct {
	Signal    string
	Timeout   time.Duration
	Mode      string
	Output    string
	Gateway   string
	LogFile   string
	Timestamp time.Time
}

func PrintGracefulShutdownBanner(w io.Writer, info ShutdownBannerInfo) {
	if w == nil {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, `  .------------------------------------------------.`)
	fmt.Fprintln(w, `  |                                                |`)
	fmt.Fprintln(w, `  |        WHOOP MORNING PRINTER / SHUTDOWN        |`)
	fmt.Fprintln(w, `  |                                                |`)
	fmt.Fprintln(w, `  |              receipt line is closed            |`)
	fmt.Fprintln(w, `  |              printer goes to sleep             |`)
	fmt.Fprintln(w, `  |                                                |`)
	fmt.Fprintln(w, `  '------------------------------------------------'`)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `          ______________________________`)
	fmt.Fprintln(w, `         /                              \`)
	fmt.Fprintln(w, `        /     GRACEFUL SHUTDOWN MODE     \`)
	fmt.Fprintln(w, `       /__________________________________\`)
	fmt.Fprintln(w, `       |                                  |`)
	fmt.Fprintln(w, `       |   [x] stop polling               |`)
	fmt.Fprintln(w, `       |   [x] cancel active context      |`)
	fmt.Fprintln(w, `       |   [x] wait worker to exit        |`)
	fmt.Fprintln(w, `       |   [x] close resources            |`)
	fmt.Fprintln(w, `       |__________________________________|`)
	fmt.Fprintln(w)

	printBannerRow(w, "signal", bannerValueOrDash(info.Signal))
	printBannerRow(w, "timeout", bannerValueOrDash(info.Timeout.String()))
	printBannerRow(w, "mode", bannerValueOrDash(info.Mode))
	printBannerRow(w, "output", bannerValueOrDash(info.Output))
	printBannerRow(w, "gateway", safeBannerURLLabel(info.Gateway))
	printBannerRow(w, "logs", bannerValueOrDash(info.LogFile))
	printBannerRow(w, "time_utc", info.Timestamp.UTC().Format(time.RFC3339))

	fmt.Fprintln(w)
	fmt.Fprintln(w, `  Press Ctrl+C again to force exit.`)
	fmt.Fprintln(w)
}

func PrintShutdownCompleteBanner(w io.Writer) {
	if w == nil {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, `  .------------------------------------------------.`)
	fmt.Fprintln(w, `  |                                                |`)
	fmt.Fprintln(w, `  |              SHUTDOWN COMPLETE                 |`)
	fmt.Fprintln(w, `  |                                                |`)
	fmt.Fprintln(w, `  |          agent stopped without panic           |`)
	fmt.Fprintln(w, `  |                                                |`)
	fmt.Fprintln(w, `  '------------------------------------------------'`)
	fmt.Fprintln(w)
}

func printBannerRow(w io.Writer, key string, value string) {
	fmt.Fprintf(w, "  %-11s %s\n", key, value)
}

func safeBannerURLLabel(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "-"
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return sanitizeBannerRawURL(rawURL)
	}

	host := strings.TrimSpace(u.Host)
	if host == "" {
		return sanitizeBannerRawURL(rawURL)
	}

	path := strings.TrimSpace(u.Path)
	if path == "" || path == "/" {
		return host
	}

	return host + path
}

func sanitizeBannerRawURL(rawURL string) string {
	if i := strings.Index(rawURL, "?"); i >= 0 {
		rawURL = rawURL[:i]
	}

	if i := strings.LastIndex(rawURL, "@"); i >= 0 {
		rawURL = rawURL[i+1:]
	}

	return bannerValueOrDash(rawURL)
}

func bannerValueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}

	return value
}
