package metrics

import (
	"fmt"
	"strings"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-rabbitmq/data"
	"github.com/newrelic/nri-rabbitmq/data/consts"
)

var vhostMetrics = []struct {
	metricName string
	state      string
	sourceType metric.SourceType
}{
	{"vhost.connectionsTotal", "total", metric.GAUGE},
	{"vhost.connectionsStarting", "starting", metric.GAUGE},
	{"vhost.connectionsTuning", "tuning", metric.GAUGE},
	{"vhost.connectionsOpening", "opening", metric.GAUGE},
	{"vhost.connectionsRunning", "running", metric.GAUGE},
	{"vhost.connectionsFlow", "flow", metric.GAUGE},
	{"vhost.connectionsBlocking", "blocking", metric.GAUGE},
	{"vhost.connectionsBlocked", "blocked", metric.GAUGE},
	{"vhost.connectionsClosing", "closing", metric.GAUGE},
	{"vhost.connectionsClosed", "closed", metric.GAUGE},
}

// CollectEntityMetrics ...
func CollectEntityMetrics(rabbitmqIntegration *integration.Integration, bindings []*data.BindingData, dataItems ...data.EntityData) {
	bindingStats := collectBindingStats(bindings)
	for _, dataItem := range dataItems {
		if entity, metricNamespace, err := dataItem.GetEntity(rabbitmqIntegration); err != nil {
			log.Error("Could not create %s entity [%s]: %v", dataItem.EntityType(), dataItem.EntityName(), err)
		} else if entity != nil {
			metricSet := entity.NewMetricSet(getSampleName(dataItem.EntityType()), metricNamespace...)
			warnIfError(metricSet.MarshalMetrics(dataItem), "Error collecting metrics for %s:%s", dataItem.EntityType(), dataItem.EntityName())

			if queue, ok := dataItem.(*data.QueueData); ok {
				populateBindingMetric(queue.Name, queue.Vhost, consts.QueueType, metricSet, bindingStats)
				queue.CollectInventory(entity)
			} else if exchange, ok := dataItem.(*data.ExchangeData); ok {
				populateBindingMetric(exchange.Name, exchange.Vhost, consts.QueueType, metricSet, bindingStats)
				exchange.CollectInventory(entity)
			}
		}
	}
}

// CollectVhostMetrics collects the metrics for VHost entities
func CollectVhostMetrics(rabbitmqIntegration *integration.Integration, vhosts []*data.VhostData, connections []*data.ConnectionData) {
	connStats := collectConnectionStats(connections)
	for _, vhost := range vhosts {
		if entity, metricNamespace, err := data.CreateEntity(rabbitmqIntegration, vhost.Name, consts.VhostType, vhost.Name); err != nil {
			log.Error("Could not create vhost entity [%s]: %v", vhost.Name, err)
		} else if entity != nil {
			metricSet := entity.NewMetricSet(getSampleName(consts.VhostType), metricNamespace...)
			for _, connStatus := range vhostMetrics {
				connKey := connKey{vhost.Name, connStatus.state}
				setMetric(metricSet, connStatus.metricName, connStats[connKey], connStatus.sourceType)
			}
		}
	}
}

func getSampleName(entityType string) string {
	return fmt.Sprintf("Rabbitmq%sSample", strings.Title(entityType))
}

func warnIfError(err error, format string, args ...interface{}) {
	if err != nil {
		log.Warn(format, append([]interface{}{err}, args...))
	}
}

func setMetric(metricSet *metric.Set, metricName string, metricValue interface{}, metricType metric.SourceType) {
	if err := metricSet.SetMetric(metricName, metricValue, metricType); err != nil {
		log.Error("There was an error when trying to set metric value: %s", err)
	}
}

func populateBindingMetric(entityName, vhost, entityType string, metricSet *metric.Set, bindingsStats map[bindingKey]int) {
	bindingKey := bindingKey{
		Vhost:      vhost,
		EntityName: entityName,
		EntityType: entityType,
	}
	setMetric(metricSet, entityType+".bindings", bindingsStats[bindingKey], metric.GAUGE)
}
