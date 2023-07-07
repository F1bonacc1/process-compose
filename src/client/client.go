package client

import (
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"net/http"
	"sync"
	"time"
)

var (
	zeroTime = time.Unix(0, 0)
)

type PcClient struct {
	address    string
	port       int
	logLength  int
	logger     *LogClient
	errMtx     sync.Mutex
	firstError time.Time
	isErrored  bool
	client     *http.Client
}

func NewClient(address string, port, logLength int) *PcClient {
	return &PcClient{
		address:    address,
		port:       port,
		logLength:  logLength,
		logger:     NewLogClient(),
		firstError: zeroTime,
		isErrored:  false,
		client:     &http.Client{},
	}

}

func (p *PcClient) ShutDownProject() {
	log.Info().Msg("client detached")
}

func (p *PcClient) IsRemote() bool {
	return true
}

func (p *PcClient) GetHostName() (string, error) {
	return getHostName(p.address, p.port)
}

func (p *PcClient) GetLogLength() int {
	return p.logLength
}

func (p *PcClient) GetLogsAndSubscribe(name string, observer pclog.LogObserver) error {
	return p.logger.ReadProcessLogs(p.address, p.port, name, p.logLength, true, observer)
}

func (p *PcClient) UnSubscribeLogger(name string, observer pclog.LogObserver) error {
	return p.logger.CloseChannel()
}

func (p *PcClient) GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PcClient) GetLexicographicProcessNames() ([]string, error) {
	names, err := GetProcessesName(p.address, p.port)
	return names, err
}

func (p *PcClient) GetProcessInfo(name string) (*types.ProcessConfig, error) {
	config, err := GetProcessInfo(p.address, p.port, name)
	return config, err
}

func (p *PcClient) GetProcessState(name string) (*types.ProcessState, error) {
	state, err := GetProcessState(p.address, p.port, name)
	return state, err
}

func (p *PcClient) GetProcessesState() (*types.ProcessesState, error) {
	return GetProcessesState(p.address, p.port)
}

func (p *PcClient) StopProcess(name string) error {
	return StopProcesses(p.address, p.port, name)
}

func (p *PcClient) StartProcess(name string) error {
	return StartProcesses(p.address, p.port, name)
}

func (p *PcClient) RestartProcess(name string) error {
	return RestartProcesses(p.address, p.port, name)
}

func (p *PcClient) ScaleProcess(name string, scale int) error {
	return ScaleProcess(p.address, p.port, name, scale)
}

func (p *PcClient) IsAlive() error {
	return p.logError(isAlive(p.address, p.port))
}

func (p *PcClient) ErrorForSecs() int {
	p.errMtx.Lock()
	defer p.errMtx.Unlock()
	if !p.isErrored {
		return 0
	}
	return int(time.Now().Sub(p.firstError).Seconds())
}

func (p *PcClient) logError(err error) error {
	p.errMtx.Lock()
	defer p.errMtx.Unlock()
	if err == nil {
		p.isErrored = false
		return nil
	}
	if !p.isErrored {
		p.isErrored = true
		p.firstError = time.Now()
	}
	return err
}
