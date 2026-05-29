package health

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type grpcChecker struct {
	address string
	service string
	timeout int
}

func (c *grpcChecker) Status() (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.timeout)*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(c.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %s: %w", c.address, err)
	}
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)
	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{
		Service: c.service,
	})
	if err != nil {
		return nil, fmt.Errorf("grpc health check %s: %w", c.address, err)
	}

	details := map[string]string{
		"status":  resp.GetStatus().String(),
		"service": c.service,
		"address": c.address,
	}

	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return details, fmt.Errorf("grpc service %q on %s status: %s", c.service, c.address, resp.GetStatus().String())
	}

	return details, nil
}
