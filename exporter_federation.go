package main

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	RegisterExporter("federation", newExporterFederation)
}

var (
	federationLabels     = []string{"cluster", "vhost", "node", "queue", "exchange", "self", "status"}
	federationLabelsKeys = []string{"vhost", "status", "node", "queue", "exchange"}
)

type exporterFederation struct {
	config      *rabbitExporterConfig
	stateMetric *prometheus.GaugeVec
}

func newExporterFederation(config *rabbitExporterConfig) Exporter {
	return exporterFederation{
		config:      config,
		stateMetric: newGaugeVec("federation_state", "A metric with a value of constant '1' for each federation in a certain state", federationLabels),
	}
}

func (e exporterFederation) Collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	e.stateMetric.Reset()

	federationData, err := getStatsInfo(*e.config, "federation-links", federationLabelsKeys)
	if err != nil {
		return err
	}

	cluster := ""
	if n, ok := ctx.Value(clusterName).(string); ok {
		cluster = n
	}
	selfNode := ""
	if n, ok := ctx.Value(nodeName).(string); ok {
		selfNode = n
	}

	for _, federation := range federationData {
		self := selfLabel(config, federation.labels["node"] == selfNode)
		e.stateMetric.WithLabelValues(cluster, federation.labels["vhost"], federation.labels["node"], federation.labels["queue"], federation.labels["exchange"], self, federation.labels["status"]).Set(1)
	}

	e.stateMetric.Collect(ch)
	return nil
}

func (e exporterFederation) Describe(ch chan<- *prometheus.Desc) {
	e.stateMetric.Describe(ch)
}
