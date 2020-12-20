package slo

import (
	monitoringv1alpha1 "github.com/kanzifucius/promethues-operator-slos/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestSimpleSLOGenerateAlertRules(t *testing.T) {
	sloDefinition := &monitoringv1alpha1.Slo{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test-ns",
		},
		Spec: monitoringv1alpha1.SloSpec{
			Objectives: monitoringv1alpha1.Objectives{
				Availability: "50",
				Latency: []monitoringv1alpha1.LatencyTarget{{
					LE:     "0.1",
					Target: "95",
				},
					{
						LE:     "0.5",
						Target: "99",
					}},

				Window: "0",
			},
			TrafficRateRecord: monitoringv1alpha1.ExprBlock{
				Expr: "sum(rate(http_total[$window]))",
			},
			ErrorRateRecord: monitoringv1alpha1.ExprBlock{
				AlertMethod: "multi-window",
				Expr:        "  sum (rate(http_requests_total{job=\"service-a\", status=\"5xx\"}[$window])) /\n        sum (rate(http_requests_total{job=\"service-a\"}[$window]))",
			},
			LatencyRecord: monitoringv1alpha1.ExprBlock{
				AlertMethod: "multi-window",
				Expr:        "   sum (rate(http_request_duration_seconds_bucket{job=\"service-a\", le=\"$le\"}[$window])) /\n        sum (rate(http_requests_total{job=\"service-a\"}[$window]))",
			},
			LatencyQuantileRecord: monitoringv1alpha1.ExprBlock{},
			Labels: map[string]string{
				"team": "test-team",
			},
			Annotations: map[string]string{
				"message":   "test",
				"link":      "test",
				"dashboard": "test",
			},
		},
	}

	alertRules, _ := GeneratePromRules(sloDefinition)
	assert.NotNil(t, alertRules, "no Prometheus rule generated")
	assert.NotEmpty(t, alertRules.Spec.Groups, "no groups for Prometheus rules")
	assert.Equal(t, len(alertRules.Spec.Groups), 4, "generated rules should have 4 groups")

}
