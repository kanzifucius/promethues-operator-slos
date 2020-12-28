# promethues-operator-slos

This operator allows for the generation of **PrometheusRule** Crds instances  see [Promethues operator](https://github.com/prometheus-operator/prometheus-operator)
for generated SLOs based on promethues metrics.

# Credits and References:
The code is makes large reuse of code and designs found in the  [globocom/slo-generator](https://github.com/prometheus-operator/prometheus-operator)
and could be considered as a port of the project to a kubernetes operator.

Currently, the operator only supports the muti-window alert options as described in the **globocom/slo-generator** documentation


# Exmaple

Sample file can can be found at [sample](samples/monitoring_v1alpha1_slo.yaml)

```
apiVersion: monitoring.kanzifucius.com/v1alpha1
kind: Slo
metadata:
  name: slo-sample
spec:
    labels:
      team: testteam
      app: kube-prometheus-stack
      release: prom-operator
      test: testings
    errorRateRecord:
      alertMethod: multi-window
      burnRate: "2"
      expr: |-
          sum (rate(http_requests_total{job="service-a", status="5xx"}[$window])) /
                sum (rate(http_requests_total{job="service-a"}[$window]))
    latencyRecord:
      buckets:
        - "777"
      alertMethod: multi-window
      expr: |-
           sum (rate(http_request_duration_seconds_bucket{job="service-a", le="$le"}[$window])) /
                sum (rate(http_requests_total{job="service-a"}[$window]))
    trafficRateRecord:
      expr: sum(rate(http_total[$window]))
    objectives:
      availability: "99.9"
      latency:
        - le: "0.1"
          target: "95"
        - le: "0.5"
          target: "99"
      window: "0"

```


The bellow will result in the following Prometheus rule bneing applied to the same namespace as the crd resource

```
apiVersion: ' monitoring.coreos.com/v1'
kind: PrometheusRule
metadata:
  name: test-servvice
  namespace: test-ns
spec:
  groups:
    - interval: 30s
      name: slo:test-servvice:short
      rules:
        - expr: sum(rate(http_total[5m]))
          labels:
            team: testteam
          record: slo:test-servvice:service_traffic:ratio_rate_5m
        - expr: |2-
              sum (rate(http_requests_total{job="service-a", status="5xx"}[5m])) /
                    sum (rate(http_requests_total{job="service-a"}[5m]))
          labels:
            team: testteam
          record: slo:test-servvice:service_errors_total:ratio_rate_5m
        - expr: sum(rate(http_total[30m]))
          labels:
            team: testteam
          record: slo:test-servvice:service_traffic:ratio_rate_30m
        - expr: |2-
              sum (rate(http_requests_total{job="service-a", status="5xx"}[30m])) /
                    sum (rate(http_requests_total{job="service-a"}[30m]))
          labels:
            team: testteam
          record: slo:test-servvice:service_errors_total:ratio_rate_30m
        - expr: sum(rate(http_total[1h]))
          labels:
            team: testteam
          record: slo:test-servvice:service_traffic:ratio_rate_1h
        - expr: |2-
              sum (rate(http_requests_total{job="service-a", status="5xx"}[1h])) /
                    sum (rate(http_requests_total{job="service-a"}[1h]))
          labels:
            team: testteam
          record: slo:test-servvice:service_errors_total:ratio_rate_1h
    - interval: 2m
      name: slo:test-servvice:medium
      rules:
        - expr: sum(rate(http_total[2h]))
          labels:
            team: testteam
          record: slo:test-servvice:service_traffic:ratio_rate_2h
        - expr: |2-
              sum (rate(http_requests_total{job="service-a", status="5xx"}[2h])) /
                    sum (rate(http_requests_total{job="service-a"}[2h]))
          labels:
            team: testteam
          record: slo:test-servvice:service_errors_total:ratio_rate_2h
        - expr: sum(rate(http_total[6h]))
          labels:
            team: testteam
          record: slo:test-servvice:service_traffic:ratio_rate_6h
        - expr: |2-
              sum (rate(http_requests_total{job="service-a", status="5xx"}[6h])) /
                    sum (rate(http_requests_total{job="service-a"}[6h]))
          labels:
            team: testteam
          record: slo:test-servvice:service_errors_total:ratio_rate_6h
    - interval: 5m
      name: slo:test-servvice:daily
      rules:
        - expr: sum(rate(http_total[1d]))
          labels:
            team: testteam
          record: slo:test-servvice:service_traffic:ratio_rate_1d
        - expr: |2-
              sum (rate(http_requests_total{job="service-a", status="5xx"}[1d])) /
                    sum (rate(http_requests_total{job="service-a"}[1d]))
          labels:
            team: testteam
          record: slo:test-servvice:service_errors_total:ratio_rate_1d
        - expr: sum(rate(http_total[3d]))
          labels:
            team: testteam
          record: slo:test-servvice:service_traffic:ratio_rate_3d
        - expr: |2-
              sum (rate(http_requests_total{job="service-a", status="5xx"}[3d])) /
                    sum (rate(http_requests_total{job="service-a"}[3d]))
          labels:
            team: testteam
          record: slo:test-servvice:service_errors_total:ratio_rate_3d
    - name: slo:test-servvice:alert
      rules:
        - alert: slo:test-servvice.errors.page
          annotations:
            namespace: test-ns
            severity: page
          expr: (slo:test-servvice:service_errors_total:ratio_rate_1h{service="test-servvice"} > (14.4 * 0.5) and slo:test-servvice:service_errors_total:ratio_rate_5m{service="test-servvice"} > (14.4 * 0.5)) or (slo:test-servvice:service_errors_total:ratio_rate_6h{service="test-servvice"} > (6 * 0.5) and slo:test-servvice:service_errors_total:ratio_rate_30m{service="test-servvice"} > (6 * 0.5))
          labels:
            namespace: test-ns
            severity: page
        - alert: slo:test-servvice.errors.ticket
          annotations:
            namespace: test-ns
            severity: ticket
          expr: (slo:test-servvice:service_errors_total:ratio_rate_1d{service="test-servvice"} > (3 * 0.5) and slo:test-servvice:service_errors_total:ratio_rate_2h{service="test-servvice"} > (3 * 0.5)) or (slo:test-servvice:service_errors_total:ratio_rate_3d{service="test-servvice"} > (1 * 0.5) and slo:test-servvice:service_errors_total:ratio_rate_6h{service="test-servvice"} > (1 * 0.5))
          labels:
            namespace: test-ns
            severity: ticket
        - alert: slo:test-servvice.latency.page
          annotations:
            namespace: test-ns
            severity: page
          expr: (slo:test-servvice:service_latency:ratio_rate_1h{le="0.1", service="test-servvice"} < 0.28 and slo:test-servvice:service_latency:ratio_rate_5m{le="0.1", service="test-servvice"} < 0.28) or (slo:test-servvice:service_latency:ratio_rate_6h{le="0.1", service="test-servvice"} < 0.7 and slo:test-servvice:service_latency:ratio_rate_30m{le="0.1", service="test-servvice"} < 0.7) or (slo:test-servvice:service_latency:ratio_rate_1h{le="0.5", service="test-servvice"} < 0.856 and slo:test-servvice:service_latency:ratio_rate_5m{le="0.5", service="test-servvice"} < 0.856) or (slo:test-servvice:service_latency:ratio_rate_6h{le="0.5", service="test-servvice"} < 0.94 and slo:test-servvice:service_latency:ratio_rate_30m{le="0.5", service="test-servvice"} < 0.94)
          labels:
            namespace: test-ns
            severity: page
        - alert: slo:test-servvice.latency.ticket
          annotations:
            namespace: test-ns
            severity: ticket
          expr: (slo:test-servvice:service_latency:ratio_rate_1d{le="0.1", service="test-servvice"} < 0.85 and slo:test-servvice:service_latency:ratio_rate_2h{le="0.1", service="test-servvice"} < 0.85) or (slo:test-servvice:service_latency:ratio_rate_3d{le="0.1", service="test-servvice"} < 0.95 and slo:test-servvice:service_latency:ratio_rate_6h{le="0.1", service="test-servvice"} < 0.95) or (slo:test-servvice:service_latency:ratio_rate_1d{le="0.5", service="test-servvice"} < 0.97 and slo:test-servvice:service_latency:ratio_rate_2h{le="0.5", service="test-servvice"} < 0.97) or (slo:test-servvice:service_latency:ratio_rate_3d{le="0.5", service="test-servvice"} < 0.99 and slo:test-servvice:service_latency:ratio_rate_6h{le="0.5", service="test-servvice"} < 0.99)
          labels:
            namespace: test-ns
            severity: ticket


```