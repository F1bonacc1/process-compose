package app

import (
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
)

type ProcOpts func(proc *Process)

func withTuiOn(isTuiEnabled bool) ProcOpts {
	return func(proc *Process) {
		proc.isTuiEnabled = isTuiEnabled
	}
}

func withGlobalEnv(globalEnv []string) ProcOpts {
	return func(proc *Process) {
		proc.globalEnv = globalEnv
	}
}

func withDotEnv(dotEnvVars map[string]string) ProcOpts {
	return func(proc *Process) {
		proc.dotEnvVars = dotEnvVars
	}
}

func withLogger(logger pclog.PcLogger) ProcOpts {
	return func(proc *Process) {
		proc.logger = logger
	}
}

func withProcConf(procConf *types.ProcessConfig) ProcOpts {
	return func(proc *Process) {
		proc.procConf = procConf
	}
}

func withProcState(procState *types.ProcessState) ProcOpts {
	return func(proc *Process) {
		proc.procState = procState
	}
}

func withProcLog(procLog *pclog.ProcessLogBuffer) ProcOpts {
	return func(proc *Process) {
		proc.logBuffer = procLog
	}
}

func withShellConfig(shellConfig command.ShellConfig) ProcOpts {
	return func(proc *Process) {
		proc.shellConfig = shellConfig
	}
}

func withPrintLogs(printLogs bool) ProcOpts {
	return func(proc *Process) {
		proc.printLogs = printLogs
	}
}

func withIsMain(isMain bool) ProcOpts {
	return func(proc *Process) {
		proc.isMain = isMain
	}
}

func withExtraArgs(extraArgs []string) ProcOpts {
	return func(proc *Process) {
		proc.extraArgs = extraArgs
	}
}

func withLogsTruncate(truncateLogs bool) ProcOpts {
	return func(proc *Process) {
		proc.truncateLogs = truncateLogs
	}
}
