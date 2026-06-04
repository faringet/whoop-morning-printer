package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	fmt.Println()
	fmt.Println(" __        ___   _  ___   ___  ____  ")
	fmt.Println(" \\ \\      / / | | |/ _ \\ / _ \\|  _ \\ ")
	fmt.Println("  \\ \\ /\\ / /| |_| | | | | | | | |_) |")
	fmt.Println("   \\ V  V / |  _  | |_| | |_| |  __/ ")
	fmt.Println("    \\_/\\_/  |_| |_|\\___/ \\___/|_|    ")
	fmt.Println()
	fmt.Println("      MORNING PRINTER / LEGACY PRINTER AGENT")
	fmt.Println()

	fmt.Println("legacy printeragent smoke check")
	fmt.Println()
	fmt.Printf("version      %s\n", version)
	fmt.Printf("commit       %s\n", commit)
	fmt.Printf("build_date   %s\n", buildDate)
	fmt.Printf("go_version   %s\n", runtime.Version())
	fmt.Printf("goos         %s\n", runtime.GOOS)
	fmt.Printf("goarch       %s\n", runtime.GOARCH)
	fmt.Printf("pid          %d\n", os.Getpid())
	fmt.Printf("time_utc     %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Println()

	fmt.Println("status       OK")
	fmt.Println("message      legacy binary started successfully")
	fmt.Println()
}
