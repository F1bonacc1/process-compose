package health

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/f1bonacc1/go-health/v2"
	"github.com/f1bonacc1/go-health/v2/checkers"
	"github.com/google/go-cmp/cmp"
)

func TestProber_getHttpChecker(t *testing.T) {
	type fields struct {
		Name       string
		Probe      Probe
		Env        []string
		OnCheckEnd func(bool, bool, string, any)
	}
	tests := []struct {
		name    string
		fields  fields
		want    health.ICheckable
		wantErr bool
	}{
		{
			name:   "Happy path: default http probe",
			fields: fields{Probe: Probe{HttpGet: &HttpProbe{}}},
			want: &checkers.HTTP{Config: &checkers.HTTPConfig{
				URL: &url.URL{
					Scheme: "http",
					Host:   "127.0.0.1",
					Path:   "/",
				},
				Method:     http.MethodGet,
				StatusCode: 200,
				Client:     &http.Client{Timeout: time.Second},
				Timeout:    time.Second,
			}},
		},
		{
			name: "Happy path: include headers in http probe",
			fields: fields{
				Name:  "http",
				Probe: Probe{HttpGet: &HttpProbe{Headers: map[string]string{"foo": "bar"}}},
			},
			want: &checkers.HTTP{
				Config: &checkers.HTTPConfig{
					URL: &url.URL{
						Scheme: "http",
						Host:   "127.0.0.1",
						Path:   "/",
					},
					Method:     http.MethodGet,
					Headers:    http.Header{"Foo": {"bar"}},
					StatusCode: 200,
					Client:     &http.Client{Timeout: time.Second},
					Timeout:    time.Second,
				},
			},
		},
		{
			name: "Happy path: include status in http probe",
			fields: fields{
				Name:  "http",
				Probe: Probe{HttpGet: &HttpProbe{StatusCode: 204}},
			},
			want: &checkers.HTTP{
				Config: &checkers.HTTPConfig{
					URL: &url.URL{
						Scheme: "http",
						Host:   "127.0.0.1",
						Path:   "/",
					},
					Method:     http.MethodGet,
					StatusCode: 204,
					Client:     &http.Client{Timeout: time.Second},
					Timeout:    time.Second,
				},
			},
		},
		{
			name: "Happy path: error codes are not allowed",
			fields: fields{
				Name:  "http",
				Probe: Probe{HttpGet: &HttpProbe{StatusCode: 404}},
			},
			want: &checkers.HTTP{
				Config: &checkers.HTTPConfig{
					URL: &url.URL{
						Scheme: "http",
						Host:   "127.0.0.1",
						Path:   "/",
					},
					Method:     http.MethodGet,
					StatusCode: 200,
					Client:     &http.Client{Timeout: time.Second},
					Timeout:    time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(tt.fields.Name, tt.fields.Probe, tt.fields.Env, tt.fields.OnCheckEnd)
			if (err != nil) != tt.wantErr {
				t.Errorf("health.New error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, err := p.getHttpChecker()
			if (err != nil) != tt.wantErr {
				t.Errorf("Prober.getHttpChecker error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("Probe.getHttpChecker diff = %s", cmp.Diff(got, tt.want))
			}
		})
	}
}
