package client

import (
	"encoding/json"
	"fmt"

	"github.com/f1bonacc1/process-compose/src/types"
)

func (p *PcClient) getDependencyGraph() (*types.DependencyGraph, error) {
	url := fmt.Sprintf("http://%s/graph", p.address)
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var graph types.DependencyGraph
	if err = json.NewDecoder(resp.Body).Decode(&graph); err != nil {
		return nil, err
	}
	graph.RebuildInternalIndices()
	return &graph, nil
}
