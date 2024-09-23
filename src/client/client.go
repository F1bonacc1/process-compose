package client

import (
	"context"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	zeroTime = time.Unix(0, 0)
)

type PcClient struct {
	address    string
	logLength  int
	logger     *LogClient
	errMtx     sync.Mutex
	firstError time.Time
	isErrored  bool
	client     *http.Client
}

func NewUdsClient(sockPath string, logLength int) *PcClient {

	udsClient := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", sockPath)
			},
		},
	}
	c := newClient("unix", udsClient, logLength)
	c.logger = NewLogClient("unix", sockPath)
	return c
}

func NewTcpClient(host string, port, logLength int) *PcClient {
	address := fmt.Sprintf("%s:%d", host, port)
	c := newClient(address, &http.Client{}, logLength)
	c.logger = NewLogClient(address, "")
	return c
}

func newClient(address string, client *http.Client, logLength int) *PcClient {
	return &PcClient{
		address:    address,
		logLength:  logLength,
		firstError: zeroTime,
		isErrored:  false,
		client:     client,
	}
}

func (p *PcClient) ShutDownProject() error {
	return p.shutDownProject()
}

func (p *PcClient) IsRemote() bool {
	return true
}

func (p *PcClient) GetHostName() (string, error) {
	return p.getHostName()
}

func (p *PcClient) GetLogLength() int {
	return p.logLength
}

func (p *PcClient) GetLogsAndSubscribe(name string, observer pclog.LogObserver) error {
	_, err := p.logger.ReadProcessLogs(name, p.logLength, true, observer)
	return err
}

func (p *PcClient) UnSubscribeLogger(name string, observer pclog.LogObserver) error {
	return p.logger.CloseChannel()
}

func (p *PcClient) GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PcClient) GetLexicographicProcessNames() ([]string, error) {
	names, err := p.GetProcessesName()
	return names, err
}

func (p *PcClient) GetProcessInfo(name string) (*types.ProcessConfig, error) {
	return p.getProcessInfo(name)
}

func (p *PcClient) GetProcessPorts(name string) (*types.ProcessPorts, error) {
	return p.getProcessPorts(name)
}

func (p *PcClient) GetProcessState(name string) (*types.ProcessState, error) {
	state, err := p.getProcessState(name)
	return state, err
}

func (p *PcClient) GetProcessesState() (*types.ProcessesState, error) {
	return p.GetRemoteProcessesState()
}

func (p *PcClient) StopProcess(name string) error {
	return p.stopProcess(name)
}

func (p *PcClient) StopProcesses(names []string) (map[string]string, error) {
	return p.stopProcesses(names)
}

func (p *PcClient) StartProcess(name string) error {
	return p.startProcess(name)
}

func (p *PcClient) RestartProcess(name string) error {
	return p.restartProcess(name)
}

func (p *PcClient) ScaleProcess(name string, scale int) error {
	return p.scaleProcess(name, scale)
}

func (p *PcClient) IsAlive() error {
	return p.logError(p.isAlive())
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

func (p *PcClient) GetProjectState(withMemory bool) (*types.ProjectState, error) {
	return p.getProjectState(withMemory)
}

func (p *PcClient) SetProcessPassword(_, _ string) error {
	return fmt.Errorf("set process password not allowed for PC client")
}

func (p *PcClient) UpdateProject(project *types.Project) (map[string]string, error) {
	return p.updateProject(project)
}

func (p *PcClient) UpdateProcess(updated *types.ProcessConfig) error {
	return p.updateProcess(updated)
}
