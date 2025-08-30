package client

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net"
    "net/http"
    "sync"
    "time"

	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
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

func (p *PcClient) GetProjectName() (string, error) {
	return p.getProjectName()
}

func (p *PcClient) GetLogLength() int {
	return p.logLength
}

func (p *PcClient) GetLogsAndSubscribe(name string, observer pclog.LogObserver) error {
	fn := func(message api.LogMessage) {
		_, _ = observer.WriteString(message.Message)
	}
	_, err := p.logger.ReadProcessLogs(name, p.logLength, true, fn)
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
	return int(time.Since(p.firstError).Seconds())
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
	return errors.New("set process password not allowed for PC client")
}

func (p *PcClient) UpdateProject(project *types.Project) (map[string]string, error) {
	return p.updateProject(project)
}

func (p *PcClient) UpdateProcess(updated *types.ProcessConfig) error {
	return p.updateProcess(updated)
}

func (p *PcClient) ReloadProject() (map[string]string, error) {
	return p.reloadProject()
}

func (p *PcClient) TruncateProcessLogs(name string) error {
	return p.truncateProcessLogs(name)
}

func (p *PcClient) UpdateProcesses(processes *types.Processes) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace", p.address)
    jsonData, err := json.Marshal(processes)
    if err != nil {
        log.Err(err).Msg("failed to marshal processes")
        return nil, err
    }
    // treat 400 as partial failure and still return the map
    return p.doMapRequest(http.MethodPut, url, bytes.NewBuffer(jsonData), "failed to update some processes")
}

// doMapRequest executes an HTTP request, expecting a JSON body with
// shape map[string]string on success (200) or partial failure (400).
// For non-200/400 responses, it decodes pcError and returns it as error.
func (p *PcClient) doMapRequest(method, url string, body io.Reader, partialErrMsg string) (map[string]string, error) {
    req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
        results := map[string]string{}
        if err = json.NewDecoder(resp.Body).Decode(&results); err != nil {
            log.Err(err).Msg("failed to decode map response")
            return nil, err
        }
        if resp.StatusCode == http.StatusBadRequest {
            if partialErrMsg == "" {
                partialErrMsg = "partial failure"
            }
            return results, errors.New(partialErrMsg)
        }
        return results, nil
    }

    var respErr pcError
    if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
        log.Err(err).Msg("failed to decode error response")
        return nil, err
    }
    return nil, errors.New(respErr.Error)
}
