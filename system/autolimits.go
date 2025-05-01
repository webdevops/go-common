package system

import (
	"log/slog"
	"runtime"

	"github.com/KimMachineGun/automemlimit/memlimit"
	humanize "github.com/dustin/go-humanize"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func AutoProcMemLimit(logger *zap.SugaredLogger) {
	// set procs
	_, err := maxprocs.Set(maxprocs.Logger(logger.Infof))
	if err != nil {
		logger.Error(err)
	}

	// set memory limit
	goMemLimit, err := memlimit.SetGoMemLimitWithOpts(
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				memlimit.FromSystem,
			),
		),
		memlimit.WithLogger(slog.New(zapslog.NewHandler(logger.Desugar().Core()))),
	)
	if err != nil {
		logger.Error(err)
	}

	goProcs := runtime.GOMAXPROCS(0)
	goMaxMem := uint64(goMemLimit) // nolint:gosec
	logger.Infof(`GO resources updated: GOMAXPROC=%v GOMEMLIMIT=%v`, goProcs, humanize.Bytes(goMaxMem))
}
