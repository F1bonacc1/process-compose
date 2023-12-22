package health

import (
	"net/url"
	"reflect"
	"testing"
)

func TestProbe_validateAndSetDefaults(t *testing.T) {
	type fields struct {
		Exec             *ExecProbe
		Http             *HttpProbe
		InitialDelay     int
		PeriodSeconds    int
		TimeoutSeconds   int
		SuccessThreshold int
		FailureThreshold int
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
				InitialDelay:     0,
				PeriodSeconds:    10,
				TimeoutSeconds:   1,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
		},
		{
			name: "InValid - Negative",
			fields: fields{
				InitialDelay:     -1,
				PeriodSeconds:    -1,
				TimeoutSeconds:   -1,
				SuccessThreshold: -1,
				FailureThreshold: -1,
			},
			want: fields{
				InitialDelay:     0,
				PeriodSeconds:    10,
				TimeoutSeconds:   1,
				SuccessThreshold: 1,
				FailureThreshold: 3,
			},
		},
		{
			name: "Valid - No Change",
			fields: fields{
				InitialDelay:     5,
				PeriodSeconds:    5,
				TimeoutSeconds:   5,
				SuccessThreshold: 5,
				FailureThreshold: 5,
			},
			want: fields{
				InitialDelay:     5,
				PeriodSeconds:    5,
				TimeoutSeconds:   5,
				SuccessThreshold: 5,
				FailureThreshold: 5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Probe{
				Exec:             tt.fields.Exec,
				Http:             tt.fields.Http,
				InitialDelay:     tt.fields.InitialDelay,
				PeriodSeconds:    tt.fields.PeriodSeconds,
				TimeoutSeconds:   tt.fields.TimeoutSeconds,
				SuccessThreshold: tt.fields.SuccessThreshold,
				FailureThreshold: tt.fields.FailureThreshold,
			}
			p.validateAndSetDefaults()
			if p.InitialDelay != tt.want.InitialDelay {
				t.Errorf("Probe.InitialDelay = %v, want %v", p.InitialDelay, tt.want.InitialDelay)
			}
			if p.PeriodSeconds != tt.want.PeriodSeconds {
				t.Errorf("Probe.PeriodSeconds = %v, want %v", p.PeriodSeconds, tt.want.PeriodSeconds)
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

func TestHttpProbe_getUrl(t *testing.T) {
	type fields struct {
		Host   string
		Path   string
		Scheme string
		Port   int
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
				Port:   0,
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
				Port:   443,
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
