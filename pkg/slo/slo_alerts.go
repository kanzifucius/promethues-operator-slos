package slo

import (
	promoperator "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"time"
)

type AlertErrorOptions struct {
	ServiceName        string
	AvailabilityTarget string
	SLOWindow          time.Duration

	Windows     []Window
	ShortWindow bool
	BurnRate    string
}

type AlertLatencyOptions struct {
	ServiceName string
	Targets     []LatencyTarget
	SLOWindow   time.Duration

	Windows     []Window
	ShortWindow bool
	BurnRate    string
}

type AlertMethod interface {
	AlertForError(*AlertErrorOptions) ([]promoperator.Rule, error)
	AlertForLatency(*AlertLatencyOptions) ([]promoperator.Rule, error)
}

var methods = map[string]AlertMethod{}

func register(method AlertMethod, name string) AlertMethod {
	methods[name] = method
	return method
}

func GetAlertMethod(name string) AlertMethod {
	return methods[name]
}

type LatencyTarget struct {
	LE     string
	Target float64
}
