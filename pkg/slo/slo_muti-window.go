package slo

import (
	"fmt"
	promoperator "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
)

type MultiWindowAlgorithm struct{}

type MultiRateErrorOpts struct {
	Rates  []MultiRateWindow
	Metric string
	Labels labels.Labels
	Value  float64
}

type MultiRateLatencyOpts struct {
	Rates   []MultiRateWindow
	Metric  string
	Label   labels.Label
	Buckets []LatencyTarget
}

type MultiRateWindow struct {
	Multiplier  float64
	LongWindow  string
	ShortWindow string
}

var multiRateWindows = map[string][]MultiRateWindow{

	"page": {
		{
			Multiplier:  14.4,
			LongWindow:  "1h",
			ShortWindow: "5m",
		},
		{
			Multiplier:  6,
			LongWindow:  "6h",
			ShortWindow: "30m",
		},
	},
	"ticket": {
		{
			Multiplier:  3,
			LongWindow:  "1d",
			ShortWindow: "2h",
		},
		{
			Multiplier:  1,
			LongWindow:  "3d",
			ShortWindow: "6h",
		},
	},
}

func (*MultiWindowAlgorithm) AlertForError(opts *AlertErrorOptions) ([]promoperator.Rule, error) {
	ratesMap := genMultiRateWindows(opts.SLOWindow, opts.ShortWindow, opts.Windows)
	var rules []promoperator.Rule

	for _, severity := range Severities {
		if _, ok := ratesMap[severity]; !ok {
			continue
		}

		AvailabilityTarget, err := strconv.ParseFloat(opts.AvailabilityTarget, 32)
		if err == nil {
			fmt.Println(err) // 3.1415927410125732
		}

		multiBurnRate := multiBurnRate(MultiRateErrorOpts{
			Rates:  ratesMap[severity],
			Metric: fmt.Sprintf("slo:%s:service_errors_total", opts.ServiceName),
			Labels: labels.New(labels.Label{Name: "service", Value: opts.ServiceName}),
			Value:  1 - AvailabilityTarget/100,
		})

		rules = append(rules, promoperator.Rule{
			Alert: "slo:" + opts.ServiceName + ".errors." + severity,
			Expr: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: multiBurnRate,
			},
			Annotations: map[string]string{
				"severity": severity,
			},
			Labels: map[string]string{
				"severity": severity,
			},
		})
	}
	return rules, nil
}

func (*MultiWindowAlgorithm) AlertForLatency(opts *AlertLatencyOptions) ([]promoperator.Rule, error) {
	ratesMap := genMultiRateWindows(opts.SLOWindow, opts.ShortWindow, opts.Windows)
	var rules []promoperator.Rule

	for _, severity := range Severities {
		if _, ok := ratesMap[severity]; !ok {
			continue
		}
		burnRate := multiBurnRateLatency(MultiRateLatencyOpts{
			Rates:   ratesMap[severity],
			Metric:  fmt.Sprintf("slo:%s:service_latency", opts.ServiceName),
			Label:   labels.Label{Name: "service", Value: opts.ServiceName},
			Buckets: opts.Targets,
		})

		rules = append(rules, promoperator.Rule{

			Alert: "slo:" + opts.ServiceName + ".latency." + severity,
			Expr: intstr.IntOrString{
				Type: intstr.String,

				StrVal: burnRate,
			},
			For: "",
			Labels: map[string]string{
				"severity": severity,
			},
			Annotations: map[string]string{
				"severity": severity,
			},
		})
	}

	return rules, nil
}

func genMultiRateWindows(SLOWindow time.Duration, shortWindow bool, windows []Window) map[string][]MultiRateWindow {
	if len(windows) == 0 {
		// Use Default multiRateWindows from SRE Book
		return multiRateWindows
	}

	mrate := map[string][]MultiRateWindow{}
	wHours := float64(SLOWindow / time.Hour)

	for _, w := range windows {
		t := float64(time.Duration(w.Duration) / time.Hour)

		burnRate := (w.Consumption / 100) / (t / wHours)
		m := MultiRateWindow{
			Multiplier: burnRate,
			LongWindow: w.Duration.String(),
		}

		if shortWindow {
			// Short window is defined as 1/12 of the long window for now
			short := time.Duration(w.Duration) / 12
			m.ShortWindow = model.Duration(short).String()
		}
		mrate[w.Notification] = append(mrate[w.Notification], m)
	}

	return mrate
}

func multiBurnRate(opts MultiRateErrorOpts) string {
	multiRateWindow := opts.Rates
	conditions := []string{}

	for _, window := range multiRateWindow {
		condition := fmt.Sprintf(`%s:ratio_rate_%s%s > (%g * %.3g)`, opts.Metric, window.LongWindow, opts.Labels.String(), window.Multiplier, opts.Value)
		if window.ShortWindow != "" {
			condition = fmt.Sprintf(`(%s and %s:ratio_rate_%s%s > (%g * %.3g))`, condition, opts.Metric, window.ShortWindow, opts.Labels.String(), window.Multiplier, opts.Value)
		}

		conditions = append(conditions, condition)
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	return strings.Join(conditions, " or ")
}

func multiBurnRateLatency(opts MultiRateLatencyOpts) string {
	multiRateWindow := opts.Rates
	var conditions []string

	for _, bucket := range opts.Buckets {
		for _, window := range multiRateWindow {

			value := (1 - ((100 - bucket.Target) / 100 * window.Multiplier))
			lbs := labels.New(opts.Label, labels.Label{Name: "le", Value: bucket.LE})

			condition := fmt.Sprintf(`%s:ratio_rate_%s%s < %.3g`, opts.Metric, window.LongWindow, lbs.String(), value)
			if window.ShortWindow != "" {
				condition = fmt.Sprintf(`(%s and %s:ratio_rate_%s%s < %.3g)`, condition, opts.Metric, window.ShortWindow, lbs.String(), value)
			}

			conditions = append(conditions, condition)
		}
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	return strings.Join(conditions, " or ")
}

func init() {
	register(&MultiWindowAlgorithm{}, "multi-window")
}
