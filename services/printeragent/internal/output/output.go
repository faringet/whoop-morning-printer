package output

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/printeragent/internal/storage"
)

const (
	ModeFile    = "file"
	ModeStdout  = "stdout"
	ModePrinter = "printer"
)

type Config struct {
	Mode string

	Dir        string
	CreateDirs bool

	TrailingBlankLines int

	PrinterName string
	CPI         int
	LPI         int

	SpoolDir       string
	KeepSpoolFiles bool
}

type Result struct {
	Destination string
	Bytes       int
}

type Printer interface {
	Print(ctx context.Context, job storage.PrintJob) (Result, error)
}

func New(log *slog.Logger, cfg Config) (Printer, error) {
	if log == nil {
		log = slog.Default()
	}

	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode == "" {
		cfg.Mode = ModeFile
	}

	if cfg.TrailingBlankLines < 0 {
		cfg.TrailingBlankLines = 0
	}

	switch cfg.Mode {
	case ModeFile:
		cfg.Dir = strings.TrimSpace(cfg.Dir)
		if cfg.Dir == "" {
			cfg.Dir = "./out/receipts"
		}

		return &FilePrinter{
			log: log.With(
				slog.String("layer", "output"),
				slog.String("module", "printeragent.output.file"),
			),
			dir:                cfg.Dir,
			createDirs:         cfg.CreateDirs,
			trailingBlankLines: cfg.TrailingBlankLines,
		}, nil

	case ModeStdout:
		return &StdoutPrinter{
			log: log.With(
				slog.String("layer", "output"),
				slog.String("module", "printeragent.output.stdout"),
			),
			trailingBlankLines: cfg.TrailingBlankLines,
		}, nil

	case ModePrinter:
		cfg.PrinterName = strings.TrimSpace(cfg.PrinterName)
		if cfg.PrinterName == "" {
			return nil, errors.New("printeragent output printer: printer_name is required")
		}

		if cfg.CPI <= 0 {
			cfg.CPI = 16
		}

		if cfg.LPI <= 0 {
			cfg.LPI = 8
		}

		cfg.SpoolDir = strings.TrimSpace(cfg.SpoolDir)
		if cfg.SpoolDir == "" {
			cfg.SpoolDir = "./out/print-spool"
		}

		return &LPPrinter{
			log: log.With(
				slog.String("layer", "output"),
				slog.String("module", "printeragent.output.printer"),
			),
			printerName:        cfg.PrinterName,
			cpi:                cfg.CPI,
			lpi:                cfg.LPI,
			spoolDir:           cfg.SpoolDir,
			createDirs:         cfg.CreateDirs,
			keepSpoolFiles:     cfg.KeepSpoolFiles,
			trailingBlankLines: cfg.TrailingBlankLines,
		}, nil

	default:
		return nil, fmt.Errorf("printeragent output: unsupported mode %q", cfg.Mode)
	}
}

type FilePrinter struct {
	log *slog.Logger

	dir        string
	createDirs bool

	trailingBlankLines int
}

func (p *FilePrinter) Print(ctx context.Context, job storage.PrintJob) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	if err := validatePrintJob(job); err != nil {
		return Result{}, err
	}

	if p.createDirs {
		if err := os.MkdirAll(p.dir, 0o755); err != nil {
			return Result{}, fmt.Errorf("printeragent output file: create dir: %w", err)
		}
	}

	fileName := buildFileName(job)
	path := filepath.Join(p.dir, fileName)

	payload := preparePayload(job.PayloadText, p.trailingBlankLines)

	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		return Result{}, fmt.Errorf("printeragent output file: write receipt: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return Result{}, fmt.Errorf("printeragent output file: stat receipt: %w", err)
	}

	p.log.Info("receipt written to file",
		slog.Int64("print_job_id", job.ID),
		slog.String("type", string(job.Type)),
		slog.String("path", path),
		slog.Int64("bytes", info.Size()),
	)

	return Result{
		Destination: path,
		Bytes:       int(info.Size()),
	}, nil
}

type StdoutPrinter struct {
	log *slog.Logger

	trailingBlankLines int
}

func (p *StdoutPrinter) Print(ctx context.Context, job storage.PrintJob) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	if err := validatePrintJob(job); err != nil {
		return Result{}, err
	}

	payload := preparePayload(job.PayloadText, p.trailingBlankLines)

	fmt.Println()
	fmt.Println("========== PRINT JOB BEGIN ==========")
	fmt.Printf("job_id: %d\n", job.ID)
	fmt.Printf("type: %s\n", job.Type)
	if job.WakePlanID != nil {
		fmt.Printf("wake_plan_id: %d\n", *job.WakePlanID)
	}
	fmt.Println("-------------------------------------")
	fmt.Print(payload)
	fmt.Println("=========== PRINT JOB END ===========")
	fmt.Println()

	p.log.Info("receipt written to stdout",
		slog.Int64("print_job_id", job.ID),
		slog.String("type", string(job.Type)),
		slog.Int("bytes", len([]byte(payload))),
	)

	return Result{
		Destination: "stdout",
		Bytes:       len([]byte(payload)),
	}, nil
}

type LPPrinter struct {
	log *slog.Logger

	printerName string

	cpi int
	lpi int

	spoolDir       string
	createDirs     bool
	keepSpoolFiles bool

	trailingBlankLines int
}

func (p *LPPrinter) Print(ctx context.Context, job storage.PrintJob) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	if err := validatePrintJob(job); err != nil {
		return Result{}, err
	}

	if p.createDirs {
		if err := os.MkdirAll(p.spoolDir, 0o755); err != nil {
			return Result{}, fmt.Errorf("printeragent output printer: create spool dir: %w", err)
		}
	}

	payload := preparePayload(job.PayloadText, p.trailingBlankLines)

	fileName := buildFileName(job)
	path := filepath.Join(p.spoolDir, fileName)

	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		return Result{}, fmt.Errorf("printeragent output printer: write spool receipt: %w", err)
	}

	args := []string{
		"-d", p.printerName,
		"-o", fmt.Sprintf("cpi=%d", p.cpi),
		"-o", fmt.Sprintf("lpi=%d", p.lpi),
		path,
	}

	cmd := exec.CommandContext(ctx, "lp", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		p.log.Error("lp print command failed",
			slog.Int64("print_job_id", job.ID),
			slog.String("type", string(job.Type)),
			slog.String("printer_name", p.printerName),
			slog.String("spool_path", path),
			slog.String("command", "lp"),
			slog.Any("args", args),
			slog.String("output", strings.TrimSpace(string(output))),
			slog.Any("err", err),
		)

		return Result{}, fmt.Errorf("printeragent output printer: lp failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	if !p.keepSpoolFiles {
		if err := os.Remove(path); err != nil {
			p.log.Warn("remove spool receipt failed",
				slog.Int64("print_job_id", job.ID),
				slog.String("spool_path", path),
				slog.Any("err", err),
			)
		}
	}

	p.log.Info("receipt sent to printer",
		slog.Int64("print_job_id", job.ID),
		slog.String("type", string(job.Type)),
		slog.String("printer_name", p.printerName),
		slog.String("spool_path", path),
		slog.Int("cpi", p.cpi),
		slog.Int("lpi", p.lpi),
		slog.Bool("keep_spool_files", p.keepSpoolFiles),
		slog.String("lp_output", strings.TrimSpace(string(output))),
		slog.Int("bytes", len([]byte(payload))),
	)

	return Result{
		Destination: "printer:" + p.printerName,
		Bytes:       len([]byte(payload)),
	}, nil
}

func validatePrintJob(job storage.PrintJob) error {
	if job.ID <= 0 {
		return errors.New("printeragent output: print_job.id must be > 0")
	}

	if strings.TrimSpace(job.PayloadText) == "" {
		return errors.New("printeragent output: payload_text is empty")
	}

	payloadType := strings.TrimSpace(job.PayloadType)
	if payloadType == "" {
		payloadType = storage.PayloadTypeTextPlain
	}

	if payloadType != storage.PayloadTypeTextPlain {
		return fmt.Errorf("printeragent output: unsupported payload_type %q", payloadType)
	}

	return nil
}

func preparePayload(payload string, trailingBlankLines int) string {
	payload = strings.ReplaceAll(payload, "\r\n", "\n")
	payload = strings.ReplaceAll(payload, "\r", "\n")
	payload = strings.TrimRight(payload, "\n")

	if trailingBlankLines > 0 {
		payload += strings.Repeat("\n", trailingBlankLines)
	}

	return payload + "\n"
}

func buildFileName(job storage.PrintJob) string {
	baseTime := job.NotBefore
	if baseTime.IsZero() {
		baseTime = time.Now().UTC()
	}

	timestamp := baseTime.UTC().Format("20060102_150405")
	jobType := sanitizeFilePart(string(job.Type))
	if jobType == "" {
		jobType = "print_job"
	}

	if job.WakePlanID != nil && *job.WakePlanID > 0 {
		return fmt.Sprintf(
			"%s_%s_wakeplan_%d_job_%d.txt",
			timestamp,
			jobType,
			*job.WakePlanID,
			job.ID,
		)
	}

	return fmt.Sprintf(
		"%s_%s_job_%d.txt",
		timestamp,
		jobType,
		job.ID,
	)
}

var invalidFilePartChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func sanitizeFilePart(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)
	value = invalidFilePartChars.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")

	return value
}
