package app

import (
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
)

type ProcOpts func(*Process)

func withTuiOn(tuiOn bool) ProcOpts {
	return func(p *Process) {
		p.isTuiEnabled = tuiOn
	}
}

func withGlobalEnv(env []string) ProcOpts {
	return func(p *Process) {
		p.globalEnv = env
	}
}

func withDotEnv(env map[string]string) ProcOpts {
	return func(p *Process) {
		p.dotEnvVars = env
	}
}

func withLogger(logger pclog.PcLogger) ProcOpts {
	return func(p *Process) {
		p.logger = logger
	}
}

func withProcConf(procConf *types.ProcessConfig) ProcOpts {
	return func(p *Process) {
		p.procConf = procConf
	}
}

func withProcState(procState *types.ProcessState) ProcOpts {
	return func(p *Process) {
		p.procState = procState
	}
}

func withProcLog(procLog *pclog.ProcessLogBuffer) ProcOpts {
	return func(p *Process) {
		p.logBuffer = procLog
	}
}

func withShellConfig(shellConfig command.ShellConfig) ProcOpts {
	return func(p *Process) {
		p.shellConfig = shellConfig
	}
}

func withPrintLogs(printLogs bool) ProcOpts {
	return func(p *Process) {
		p.printLogs = printLogs
	}
}

func withIsMain(isMain bool) ProcOpts {
	return func(p *Process) {
		p.isMain = isMain
	}
}

func withExtraArgs(extraArgs []string) ProcOpts {
	return func(p *Process) {
		p.extraArgs = extraArgs
	}
}

func withLogsTruncate(truncate bool) ProcOpts {
	return func(p *Process) {
		p.truncateLogs = truncate
	}
}

func withRefRate(refRate time.Duration) ProcOpts {
	return func(p *Process) {
		p.refRate = refRate
	}
}

func withRecursiveMetrics(recursive bool) ProcOpts {
	return func(p *Process) {
		p.withRecursiveMetrics = recursive
	}
}

func withProcessTree(tree *ProcessTree) ProcOpts {
	return func(p *Process) {
		p.processTree = tree
	}
}
