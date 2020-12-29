package slo

import (
	"fmt"
	monitoringv1alpha1 "github.com/kanzifucius/promethues-operator-slos/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"log"
	"strconv"
	"strings"
	"time"

	promoperator "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus/common/model"
)

var (
	// Severities list of available severities: page and ticket
	Severities = []string{
		"page",
		"ticket",
	}
)

type Window struct {
	Duration     model.Duration
	Consumption  float64
	Notification string
}

var quantiles = []struct {
	name     string
	quantile float64
}{
	{
		name:     "p50",
		quantile: 0.5,
	},
	{
		name:     "p95",
		quantile: 0.95,
	},
	{
		name:     "p99",
		quantile: 0.99,
	},
}

func GeneratePromRules(sloDefinition *monitoringv1alpha1.Slo) (*promoperator.PrometheusRule, error) {

	var Groups []promoperator.RuleGroup

	ruleGroupRules, err := generateGroupRules(sloDefinition)
	if err != nil {
		return nil, err
	}
	Groups = append(Groups, ruleGroupRules...)

	ruleAlerts, err := generateAlertRules(sloDefinition)
	if err != nil {
		return nil, err
	}
	Groups = append(Groups, promoperator.RuleGroup{
		Name:  "slo:" + santizeString(sloDefinition.Name) + ":alert",
		Rules: ruleAlerts,
	})

	prometheusRule := &promoperator.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			Kind:       promoperator.PrometheusRuleKind,
			APIVersion: " monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      sloDefinition.Name,
			Namespace: sloDefinition.Namespace,
			Labels:    sloDefinition.Labels,
		},
		Spec: promoperator.PrometheusRuleSpec{Groups: Groups},
	}

	// Set Memcached instance as the owner and controller

	return prometheusRule, nil
}

func generateAlertRules(sloDefinition *monitoringv1alpha1.Slo) ([]promoperator.Rule, error) {

	var alertRules []promoperator.Rule

	if sloDefinition.Spec.ErrorRateRecord.AlertMethod != "" {
		errorMethod := GetAlertMethod(sloDefinition.Spec.ErrorRateRecord.AlertMethod)
		if errorMethod == nil {
			log.Panicf("alertMethod %s is not valid", sloDefinition.Spec.ErrorRateRecord.AlertMethod)
		}

		var Windows []Window
		for _, recordWindow := range sloDefinition.Spec.ErrorRateRecord.Windows {
			recWindowDuration, err := model.ParseDuration(recordWindow.Duration)
			if err != nil {
				return nil, fmt.Errorf("failed to convert %s to float", recordWindow.Duration)
			}

			recWindowConsumption, err := strconv.ParseFloat(recordWindow.Consumption, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to convert %s to float", recordWindow.Consumption)
			}

			Windows = append(Windows, Window{

				Duration:     recWindowDuration,
				Consumption:  recWindowConsumption,
				Notification: recordWindow.Notification,
			})
		}

		objectivesWindow, err := time.ParseDuration(sloDefinition.Spec.Objectives.Window)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s to duration", sloDefinition.Spec.Objectives.Window)
		}

		errorRules, err := errorMethod.AlertForError(&AlertErrorOptions{
			ServiceName:        santizeString(sloDefinition.Name),
			AvailabilityTarget: sloDefinition.Spec.Objectives.Availability,
			SLOWindow:          objectivesWindow,
			ShortWindow:        sloDefinition.Spec.ErrorRateRecord.GetShortWindow(),
			Windows:            Windows,
			BurnRate:           sloDefinition.Spec.ErrorRateRecord.BurnRate,
		})
		if err != nil {
			log.Panicf("Could not generate alert, err: %s", err.Error())
		}
		alertRules = append(alertRules, errorRules...)
	}

	if sloDefinition.Spec.LatencyRecord.AlertMethod != "" {
		latencyMethod := GetAlertMethod(sloDefinition.Spec.LatencyRecord.AlertMethod)
		if latencyMethod == nil {
			log.Panicf("alertMethod %s is not valid", sloDefinition.Spec.LatencyRecord.AlertMethod)
		}

		if sloDefinition.Spec.Objectives.Latency != nil {
			var LatencyTargets []LatencyTarget
			for _, record := range sloDefinition.Spec.Objectives.Latency {

				target, err := strconv.ParseFloat(record.Target, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to convert %s to float", record.Target)

				}

				LatencyTargets = append(LatencyTargets, LatencyTarget{
					LE:     record.LE,
					Target: target,
				})
			}

			var Windows []Window
			for _, recordWindow := range sloDefinition.Spec.LatencyRecord.Windows {

				recWindowDuration, err := model.ParseDuration(recordWindow.Duration)
				if err != nil {
					return nil, fmt.Errorf("failed to convert %s to duration", recordWindow.Duration)
				}

				recWindowConsumption, err := strconv.ParseFloat(recordWindow.Consumption, 64)
				if err != nil {
					return nil, fmt.Errorf("failed to convert %s to float", recordWindow.Consumption)
				}

				Windows = append(Windows, Window{
					Duration:     recWindowDuration,
					Consumption:  recWindowConsumption,
					Notification: recordWindow.Notification,
				})
			}

			objectivesWindow, err := time.ParseDuration(sloDefinition.Spec.Objectives.Window)
			if err != nil {
				return nil, fmt.Errorf("failed to convert %s to duration", sloDefinition.Spec.Objectives.Window)
			}

			latencyRules, err := latencyMethod.AlertForLatency(&AlertLatencyOptions{
				ServiceName: santizeString(sloDefinition.Name),
				Targets:     LatencyTargets,
				SLOWindow:   objectivesWindow,
				ShortWindow: sloDefinition.Spec.LatencyRecord.GetShortWindow(),
				Windows:     Windows,
				BurnRate:    sloDefinition.Spec.ErrorRateRecord.BurnRate,
			})
			if err != nil {
				log.Panicf("Could not generate alert, err: %s", err.Error())
			}
			alertRules = append(alertRules, latencyRules...)
		}
	}

	for _, rule := range alertRules {
		fillMetadata(&rule, sloDefinition)
	}

	return alertRules, nil
}

func fillMetadata(rule *promoperator.Rule, definition *monitoringv1alpha1.Slo) {
	rule.Labels["namespace"] = definition.Namespace
	for label, value := range definition.Labels {
		rule.Labels[label] = value
	}

	rule.Annotations["namespace"] = definition.Namespace

}

func generateGroupRules(slo *monitoringv1alpha1.Slo) ([]promoperator.RuleGroup, error) {
	var rules []promoperator.RuleGroup

	var latencyBuckets []string
	if len(slo.Spec.LatencyRecord.Buckets) > 0 {
		for _, bucket := range slo.Spec.LatencyRecord.Buckets {
			latencyBuckets = append(latencyBuckets, bucket)
		}

	}

	for _, sample := range monitoringv1alpha1.DefaultSamples {

		ruleGroup := promoperator.RuleGroup{
			Name:     fmt.Sprintf("slo:%s:%s", slo.Name, sample.Name),
			Interval: sample.Interval,
			Rules:    []promoperator.Rule{},
		}

		for _, bucket := range sample.Buckets {
			ruleGroup.Rules = append(ruleGroup.Rules, generateRules(bucket, latencyBuckets, slo)...)
		}

		if len(ruleGroup.Rules) > 0 {
			rules = append(rules, ruleGroup)
		}
	}

	return rules, nil
}

func Labelslabels(slo *monitoringv1alpha1.Slo) map[string]string {
	labels := make(map[string]string)
	labels["service"] = slo.Name
	for key, value := range slo.Labels {
		labels[key] = value
	}
	return labels
}

func generateRules(bucket string, latencyBuckets []string, sloDefinition *monitoringv1alpha1.Slo) []promoperator.Rule {
	var rules []promoperator.Rule
	if sloDefinition.Spec.TrafficRateRecord.Expr != "" {
		trafficRateRecord := promoperator.Rule{
			Record: fmt.Sprintf("slo:%s:service_traffic:ratio_rate_%s", santizeString(sloDefinition.Name), bucket),
			Expr:   intstr.IntOrString{Type: intstr.String, StrVal: sloDefinition.Spec.TrafficRateRecord.ComputeExpr(bucket, "")},
			Labels: sloDefinition.Spec.Labels,
		}

		rules = append(rules, trafficRateRecord)
	}

	if sloDefinition.Spec.ErrorRateRecord.Expr != "" {
		errorRateRecord := promoperator.Rule{
			Record: fmt.Sprintf("slo:%s:service_errors_total:ratio_rate_%s", santizeString(sloDefinition.Name), bucket),
			Expr:   intstr.IntOrString{Type: intstr.String, StrVal: sloDefinition.Spec.ErrorRateRecord.ComputeExpr(bucket, "")},
			Labels: sloDefinition.Spec.Labels,
		}

		rules = append(rules, errorRateRecord)
	}

	if sloDefinition.Spec.LatencyQuantileRecord.Expr != "" {
		for _, quantile := range quantiles {
			latencyQuantileRecord := promoperator.Rule{
				Record: fmt.Sprintf("slo:%s:service_latency:%s_%s", santizeString(sloDefinition.Name), quantile.name, bucket),
				Expr:   intstr.IntOrString{Type: intstr.String, StrVal: sloDefinition.Spec.LatencyQuantileRecord.ComputeQuantile(bucket, quantile.quantile)},
				Labels: sloDefinition.Spec.Labels,
			}

			rules = append(rules, latencyQuantileRecord)
		}
	}

	if sloDefinition.Spec.LatencyRecord.Expr != "" {
		for _, latencyBucket := range latencyBuckets {
			latencyRateRecord := promoperator.Rule{
				Record: fmt.Sprintf("slo:%s:service_latency:ratio_rate_%s", santizeString(sloDefinition.Name), bucket),
				Expr:   intstr.IntOrString{Type: intstr.String, StrVal: sloDefinition.Spec.LatencyRecord.ComputeExpr(bucket, latencyBucket)},
				Labels: sloDefinition.Spec.Labels,
			}

			latencyRateRecord.Labels["le"] = latencyBucket

			rules = append(rules, latencyRateRecord)
		}
	}

	return rules
}

func santizeString(name string) string {

	return strings.Replace(name, "-", "_", -1)

}
