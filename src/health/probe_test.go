package health

import (
	"net/url"
	"reflect"
	"testing"
)

func TestProbe_validateAndSetDefaults(t *testing.T) {
	type fields struct {
		Exec                    *ExecProbe
		HttpGet                 *HttpProbe
		InitialDelay            int
		PeriodSeconds           int
		StartupPeriodSeconds    int
		UnhealthyPeriodSeconds  int
		FailAfterSeconds        int
		StartupFailAfterSeconds int
		TimeoutSeconds          int
		SuccessThreshold        int
		FailureThreshold        int
	}
	tests := []struct {
		name   string
		fields fields
		want   fields
	}{
		{
			name:   "InValid - Empty",
			fields: fields{},
			want: fields{
				InitialDelay:            0,
				PeriodSeconds:           10,
				StartupPeriodSeconds:    1,
				UnhealthyPeriodSeconds:  10,
				FailAfterSeconds:        20,
				StartupFailAfterSeconds: 20,
				TimeoutSeconds:          1,
				SuccessThreshold:        1,
				FailureThreshold:        3,
			},
		},
		{
			name: "InValid - Negative",
			fields: fields{
				InitialDelay:           -1,
				PeriodSeconds:          -1,
				StartupPeriodSeconds:   -1,
				UnhealthyPeriodSeconds: -1,
				TimeoutSeconds:         -1,
				SuccessThreshold:       -1,
				FailureThreshold:       -1,
			},
			want: fields{
				InitialDelay:            0,
				PeriodSeconds:           10,
				StartupPeriodSeconds:    1,
				UnhealthyPeriodSeconds:  10,
				FailAfterSeconds:        20,
				StartupFailAfterSeconds: 20,
				TimeoutSeconds:          1,
				SuccessThreshold:        1,
				FailureThreshold:        3,
			},
		},
		{
			name: "Valid - No Change",
			fields: fields{
				InitialDelay:            5,
				PeriodSeconds:           5,
				StartupPeriodSeconds:    5,
				UnhealthyPeriodSeconds:  5,
				FailAfterSeconds:        5,
				StartupFailAfterSeconds: 5,
				TimeoutSeconds:          5,
				SuccessThreshold:        5,
				FailureThreshold:        5,
			},
			want: fields{
				InitialDelay:            5,
				PeriodSeconds:           5,
				StartupPeriodSeconds:    5,
				UnhealthyPeriodSeconds:  5,
				FailAfterSeconds:        5,
				StartupFailAfterSeconds: 5,
				TimeoutSeconds:          5,
				SuccessThreshold:        5,
				FailureThreshold:        5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{
				Exec:                    tt.fields.Exec,
				HttpGet:                 tt.fields.HttpGet,
				InitialDelay:            tt.fields.InitialDelay,
				PeriodSeconds:           tt.fields.PeriodSeconds,
				StartupPeriodSeconds:    tt.fields.StartupPeriodSeconds,
				UnhealthyPeriodSeconds:  tt.fields.UnhealthyPeriodSeconds,
				FailAfterSeconds:        tt.fields.FailAfterSeconds,
				StartupFailAfterSeconds: tt.fields.StartupFailAfterSeconds,
				TimeoutSeconds:          tt.fields.TimeoutSeconds,
				SuccessThreshold:        tt.fields.SuccessThreshold,
				FailureThreshold:        tt.fields.FailureThreshold,
			}
			p.ValidateAndSetDefaults()
			if p.InitialDelay != tt.want.InitialDelay {
				t.Errorf("Probe.InitialDelay = %v, want %v", p.InitialDelay, tt.want.InitialDelay)
			}
			if p.PeriodSeconds != tt.want.PeriodSeconds {
				t.Errorf("Probe.PeriodSeconds = %v, want %v", p.PeriodSeconds, tt.want.PeriodSeconds)
			}
			if p.StartupPeriodSeconds != tt.want.StartupPeriodSeconds {
				t.Errorf("Probe.StartupPeriodSeconds = %v, want %v", p.StartupPeriodSeconds, tt.want.StartupPeriodSeconds)
			}
			if p.UnhealthyPeriodSeconds != tt.want.UnhealthyPeriodSeconds {
				t.Errorf("Probe.UnhealthyPeriodSeconds = %v, want %v", p.UnhealthyPeriodSeconds, tt.want.UnhealthyPeriodSeconds)
			}
			if p.FailAfterSeconds != tt.want.FailAfterSeconds {
				t.Errorf("Probe.FailAfterSeconds = %v, want %v", p.FailAfterSeconds, tt.want.FailAfterSeconds)
			}
			if p.StartupFailAfterSeconds != tt.want.StartupFailAfterSeconds {
				t.Errorf("Probe.StartupFailAfterSeconds = %v, want %v", p.StartupFailAfterSeconds, tt.want.StartupFailAfterSeconds)
			}
			if p.SuccessThreshold != tt.want.SuccessThreshold {
				t.Errorf("Probe.SuccessThreshold = %v, want %v", p.SuccessThreshold, tt.want.SuccessThreshold)
			}
			if p.TimeoutSeconds != tt.want.TimeoutSeconds {
				t.Errorf("Probe.TimeoutSeconds = %v, want %v", p.TimeoutSeconds, tt.want.TimeoutSeconds)
			}
			if p.FailureThreshold != tt.want.FailureThreshold {
				t.Errorf("Probe.FailureThreshold = %v, want %v", p.FailureThreshold, tt.want.FailureThreshold)
			}
		})
	}
}

// TestProbe_pollPeriodsTrackExplicitPeriod covers what the defaults table above
// cannot: every case there leaves PeriodSeconds resolving to 10, so it cannot
// distinguish "unhealthy follows period_seconds" from "unhealthy is hardcoded
// to 10", nor "explicit value kept" from "coincidentally equal to the period".
func TestProbe_pollPeriodsTrackExplicitPeriod(t *testing.T) {
	tests := []struct {
		name                       string
		probe                      Probe
		wantStartupPeriodSeconds   int
		wantUnhealthyPeriodSeconds int
	}{
		{
			name:                       "Unset - unhealthy falls back to an explicit period",
			probe:                      Probe{PeriodSeconds: 30},
			wantStartupPeriodSeconds:   1,
			wantUnhealthyPeriodSeconds: 30,
		},
		{
			name:                       "Negative - unhealthy falls back to the explicit period",
			probe:                      Probe{PeriodSeconds: 30, StartupPeriodSeconds: -1, UnhealthyPeriodSeconds: -1},
			wantStartupPeriodSeconds:   1,
			wantUnhealthyPeriodSeconds: 30,
		},
		{
			name:                       "Valid - explicit values are kept, not replaced by the period",
			probe:                      Probe{PeriodSeconds: 30, StartupPeriodSeconds: 5, UnhealthyPeriodSeconds: 5},
			wantStartupPeriodSeconds:   5,
			wantUnhealthyPeriodSeconds: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.probe
			p.ValidateAndSetDefaults()
			if p.StartupPeriodSeconds != tt.wantStartupPeriodSeconds {
				t.Errorf("Probe.StartupPeriodSeconds = %v, want %v", p.StartupPeriodSeconds, tt.wantStartupPeriodSeconds)
			}
			if p.UnhealthyPeriodSeconds != tt.wantUnhealthyPeriodSeconds {
				t.Errorf("Probe.UnhealthyPeriodSeconds = %v, want %v", p.UnhealthyPeriodSeconds, tt.wantUnhealthyPeriodSeconds)
			}
		})
	}
}

func TestProbe_failAfterDefaults(t *testing.T) {
	tests := []struct {
		name                 string
		probe                Probe
		wantFailAfter        int
		wantStartupFailAfter int
	}{
		{"derived from defaults", Probe{}, 20, 20},
		{"period only defaults to two periods", Probe{PeriodSeconds: 4}, 8, 8},
		{"period only preserves legacy timing", Probe{PeriodSeconds: 30}, 60, 60},
		{"derived from explicit threshold and period", Probe{PeriodSeconds: 4, FailureThreshold: 5}, 16, 16},
		{"startup follows explicit fail_after", Probe{FailAfterSeconds: 7}, 7, 7},
		{"explicit budgets are kept", Probe{FailAfterSeconds: 7, StartupFailAfterSeconds: 3}, 7, 3},
		{"threshold 1 derives immediate give-up", Probe{FailureThreshold: 1}, 0, 0},
		{"negative budgets derive/default", Probe{FailAfterSeconds: -1, StartupFailAfterSeconds: -1}, 20, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.probe
			p.ValidateAndSetDefaults()
			if p.FailAfterSeconds != tt.wantFailAfter {
				t.Errorf("Probe.FailAfterSeconds = %v, want %v", p.FailAfterSeconds, tt.wantFailAfter)
			}
			if p.StartupFailAfterSeconds != tt.wantStartupFailAfter {
				t.Errorf("Probe.StartupFailAfterSeconds = %v, want %v", p.StartupFailAfterSeconds, tt.wantStartupFailAfter)
			}
		})
	}
}

func TestHttpProbe_getUrl(t *testing.T) {
	type fields struct {
		Host   string
		Path   string
		Scheme string
		Port   string
	}
	tUrl, _ := url.Parse("http://google.com/isAlive")
	tsUrl, _ := url.Parse("https://google.com:443/isAlive")
	tests := []struct {
		name    string
		fields  fields
		want    *url.URL
		wantErr bool
	}{
		{
			name: "Valid URL - No Port",
			fields: fields{
				Host:   "google.com",
				Path:   "/isAlive",
				Scheme: "http",
				Port:   "0",
			},
			want:    tUrl,
			wantErr: false,
		},
		{
			name: "Valid URL - With Port",
			fields: fields{
				Host:   "google.com",
				Path:   "/isAlive",
				Scheme: "https",
				Port:   "443",
			},
			want:    tsUrl,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HttpProbe{
				Host:   tt.fields.Host,
				Path:   tt.fields.Path,
				Scheme: tt.fields.Scheme,
				Port:   tt.fields.Port,
			}
			h.validateAndSetHttpDefaults()
			got, err := h.getUrl()
			if (err != nil) != tt.wantErr {
				t.Errorf("HttpProbe.getUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HttpProbe.getUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}
