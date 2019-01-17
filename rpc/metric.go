package rpc

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/intel-go/fastjson"
	"github.com/osamingo/jsonrpc"

	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"

	"github.com/icon-project/goloop/module"
)

var (
	height   = stats.Int64("consensus_status_height", "height", stats.UnitDimensionless)
	round    = stats.Int64("consensus_status_round", "round", stats.UnitDimensionless)
	proposer = stats.Int64("consensus_status_proposer", "proposer", stats.UnitDimensionless)
)

var hostname, _ = tag.NewKey("hostname")

func promethusExporter(cs module.Consensus) *prometheus.Exporter {

	// prometheus
	pe, err := prometheus.NewExporter(prometheus.Options{
		Namespace: "demo",
	})

	if err != nil {
		log.Printf("Failed to create Prometheus exporter: %+v", err)
	}

	view.RegisterExporter(pe)

	if err = view.Register(
		&view.View{
			Name:        "consensus_status_height",
			Description: "height",
			Measure:     height,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{hostname},
		},
		&view.View{
			Name:        "consensus_status_round",
			Description: "round",
			Measure:     round,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{hostname},
		},
		&view.View{
			Name:        "consensus_status_proposer",
			Description: "proposer",
			Measure:     proposer,
			Aggregation: view.LastValue(),
			TagKeys:     []tag.Key{hostname},
		},
	); err != nil {
		log.Printf("Cannot register the view: %+v", err)
	}

	// Set reporting period to report data at every second.
	view.SetReportingPeriod(1000 * time.Millisecond)

	// Record some data points...
	go func() {
		// wait for consensus initializing
		// time.Sleep(5000 * time.Millisecond)

		ctx, err := tag.New(context.Background(), tag.Insert(hostname, os.Getenv("NODE_NAME")))
		if err != nil {
			log.Fatalf("Fail insert tag: %+v", err)
		}

		for {
			status := cs.GetStatus()
			leader := 0
			if status.Proposer {
				leader = 1
			}
			stats.Record(ctx, height.M(status.Height), round.M(int64(status.Round)), proposer.M(int64(leader)))
			<-time.After(1000 * time.Millisecond)
		}
	}()

	return pe
}

func jaegerExporter() {

	// jaeger
	agentEndpointURI := "localhost:6831"
	// collectorEndpointURI := "http://localhost:14268"

	je, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint: agentEndpointURI,
		// CollectorEndpoint: collectorEndpointURI,
		Process: jaeger.Process{
			ServiceName: "json-rpc",
		},
	})

	if err != nil {
		log.Fatalf("Failed to create the Jaeger exporter: %+v", err)
	}

	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	// And now finally register it as a Trace Exporter
	trace.RegisterExporter(je)
}

func statusMethodRepository(cs module.Consensus) *jsonrpc.MethodRepository {

	status := jsonrpc.NewMethodRepository()

	err := status.RegisterMethod("getConsensusStatus", consensusStatusHandler{cs: cs}, nil, nil)
	if err != nil {
		log.Fatalf("Failed to register method : %+v", err)
	}

	return status
}

// getLastBlock
type consensusStatusHandler struct {
	cs module.Consensus
}

func (h consensusStatusHandler) ServeJSONRPC(c context.Context, params *fastjson.RawMessage) (interface{}, *jsonrpc.Error) {

	status := h.cs.GetStatus()

	return status, nil
}
