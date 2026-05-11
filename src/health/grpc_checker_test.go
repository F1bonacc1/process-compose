package health

import (
	"net"
	"testing"

	"github.com/f1bonacc1/process-compose/src/command"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/health"
)

func TestProber_getGrpcChecker_DefaultHost(t *testing.T) {
	probe := Probe{Grpc: &GrpcProbe{Port: "50051"}}
	p, err := New("default host", probe, nil, *command.DefaultShellConfig(), nil)
	if err != nil {
		t.Fatalf("health.New error = %v", err)
	}
	checker, err := p.getGrpcChecker()
	if err != nil {
		t.Fatalf("getGrpcChecker error = %v", err)
	}
	gc, ok := checker.(*grpcChecker)
	if !ok {
		t.Fatalf("expected *grpcChecker, got %T", checker)
	}
	if gc.address != "127.0.0.1:50051" {
		t.Errorf("address = %q, want %q", gc.address, "127.0.0.1:50051")
	}
	if gc.service != "" {
		t.Errorf("service = %q, want %q", gc.service, "")
	}
}

func TestProber_getGrpcChecker_CustomHostAndService(t *testing.T) {
	probe := Probe{Grpc: &GrpcProbe{Host: "10.0.0.1", Port: "9090", Service: "my.service.v1"}}
	p, err := New("custom host and service", probe, nil, *command.DefaultShellConfig(), nil)
	if err != nil {
		t.Fatalf("health.New error = %v", err)
	}
	checker, err := p.getGrpcChecker()
	if err != nil {
		t.Fatalf("getGrpcChecker error = %v", err)
	}
	gc, ok := checker.(*grpcChecker)
	if !ok {
		t.Fatalf("expected *grpcChecker, got %T", checker)
	}
	if gc.address != "10.0.0.1:9090" {
		t.Errorf("address = %q, want %q", gc.address, "10.0.0.1:9090")
	}
	if gc.service != "my.service.v1" {
		t.Errorf("service = %q, want %q", gc.service, "my.service.v1")
	}
}

func TestGrpcChecker_Status_Serving(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("my.service", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(srv, healthServer)

	go srv.Serve(lis)
	defer srv.GracefulStop()

	tests := []struct {
		name    string
		service string
		wantErr bool
	}{
		{
			name:    "default service (empty)",
			service: "",
			wantErr: false,
		},
		{
			name:    "named service",
			service: "my.service",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &grpcChecker{
				address: lis.Addr().String(),
				service: tt.service,
				timeout: 5,
			}
			details, err := checker.Status()
			if (err != nil) != tt.wantErr {
				t.Errorf("Status() error = %v, wantErr %v", err, tt.wantErr)
			}
			detailsMap, ok := details.(map[string]string)
			if !ok {
				t.Fatalf("expected map[string]string, got %T", details)
			}
			if detailsMap["status"] != "SERVING" {
				t.Errorf("status = %q, want SERVING", detailsMap["status"])
			}
		})
	}
}

func TestGrpcChecker_Status_NotServing(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	healthpb.RegisterHealthServer(srv, healthServer)

	go srv.Serve(lis)
	defer srv.GracefulStop()

	checker := &grpcChecker{
		address: lis.Addr().String(),
		service: "",
		timeout: 5,
	}
	_, err = checker.Status()
	if err == nil {
		t.Error("expected error for NOT_SERVING status, got nil")
	}
}

func TestGrpcChecker_Status_Unreachable(t *testing.T) {
	checker := &grpcChecker{
		address: "127.0.0.1:1",
		service: "",
		timeout: 1,
	}
	_, err := checker.Status()
	if err == nil {
		t.Error("expected error for unreachable server, got nil")
	}
}
