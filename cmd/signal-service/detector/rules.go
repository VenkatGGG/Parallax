package detector

import (
	commonv1 "github.com/microcloud/gen/go/common/v1"
	opsv1 "github.com/microcloud/gen/go/ops/v1"
)

// Rule defines a detection rule
type Rule struct {
	Name          string
	MetricName    string
	Operator      string
	Threshold     float64
	WindowSeconds int
	Severity      commonv1.IncidentSeverity
}

// DefaultRules returns the default detection rules
func DefaultRules() []Rule {
	return []Rule{
		{
			Name:          "high_error_rate",
			MetricName:    "error_rate_percent",
			Operator:      "gt",
			Threshold:     5.0,
			WindowSeconds: 30,
			Severity:      commonv1.IncidentSeverity_INCIDENT_SEVERITY_WARNING,
		},
		{
			Name:          "critical_error_rate",
			MetricName:    "error_rate_percent",
			Operator:      "gt",
			Threshold:     10.0,
			WindowSeconds: 15,
			Severity:      commonv1.IncidentSeverity_INCIDENT_SEVERITY_CRITICAL,
		},
		{
			Name:          "high_cpu_usage",
			MetricName:    "cpu_usage_percent",
			Operator:      "gt",
			Threshold:     85.0,
			WindowSeconds: 60,
			Severity:      commonv1.IncidentSeverity_INCIDENT_SEVERITY_WARNING,
		},
		{
			Name:          "critical_cpu_usage",
			MetricName:    "cpu_usage_percent",
			Operator:      "gt",
			Threshold:     95.0,
			WindowSeconds: 30,
			Severity:      commonv1.IncidentSeverity_INCIDENT_SEVERITY_CRITICAL,
		},
		{
			Name:          "high_memory_usage",
			MetricName:    "memory_usage_percent",
			Operator:      "gt",
			Threshold:     90.0,
			WindowSeconds: 60,
			Severity:      commonv1.IncidentSeverity_INCIDENT_SEVERITY_WARNING,
		},
		{
			Name:          "high_latency",
			MetricName:    "latency_p99_ms",
			Operator:      "gt",
			Threshold:     500.0,
			WindowSeconds: 30,
			Severity:      commonv1.IncidentSeverity_INCIDENT_SEVERITY_WARNING,
		},
	}
}

// ToProto converts a Rule to proto format
func (r Rule) ToProto() *opsv1.DetectionRule {
	return &opsv1.DetectionRule{
		Name:          r.Name,
		MetricName:    r.MetricName,
		Operator:      r.Operator,
		Threshold:     r.Threshold,
		WindowSeconds: int32(r.WindowSeconds),
		Severity:      r.Severity,
	}
}

// Evaluate checks if a value breaches the rule threshold
func (r Rule) Evaluate(value float64) bool {
	switch r.Operator {
	case "gt":
		return value > r.Threshold
	case "gte":
		return value >= r.Threshold
	case "lt":
		return value < r.Threshold
	case "lte":
		return value <= r.Threshold
	case "eq":
		return value == r.Threshold
	default:
		return false
	}
}
