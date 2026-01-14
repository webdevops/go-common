package system

import (
	"log/slog"
	"runtime"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
	humanize "github.com/dustin/go-humanize"
	"go.uber.org/automaxprocs/maxprocs"
)

func AutoProcMemLimit(logger *slog.Logger) {
	logger.Info(`detecting system resources`)

	// set procs
	_, err := maxprocs.Set(maxprocs.Logger(logger.Debug))
	if err != nil {
		logger.Error(`failed to set GOMAXPROCS`, "error", err)
	}

	// set memory limit
	goMemLimit, err := memlimit.SetGoMemLimitWithOpts(
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				memlimit.FromSystem,
			),
		),
		memlimit.WithLogger(logger),
		memlimit.WithRefreshInterval(1*time.Minute),
	)
	if err != nil {
		logger.Error(`failed to set GOMEMLIMIT`, "error", err)
	}

	goProcs := runtime.GOMAXPROCS(0)
	goMaxMem := uint64(goMemLimit) // nolint:gosec
	logger.Info(`GO resources updated`, slog.Int("GOMAXPROC", goProcs), slog.String("GOMEMLIMIT", humanize.Bytes(goMaxMem)))
}
