package testutil

import (
	"github.com/geekflow/straw/internal"
	"github.com/geekflow/straw/metric"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestRequireMetricEqual(t *testing.T) {
	tests := []struct {
		name string
		got  internal.Metric
		want internal.Metric
	}{
		{
			name: "equal metrics should be equal",
			got: func() internal.Metric {
				m, _ := metric.New(
					"test",
					map[string]string{
						"t1": "v1",
						"t2": "v2",
					},
					map[string]interface{}{
						"f1": 1,
						"f2": 3.14,
						"f3": "v3",
					},
					time.Unix(0, 0),
				)
				return m
			}(),
			want: func() internal.Metric {
				m, _ := metric.New(
					"test",
					map[string]string{
						"t1": "v1",
						"t2": "v2",
					},
					map[string]interface{}{
						"f1": int64(1),
						"f2": 3.14,
						"f3": "v3",
					},
					time.Unix(0, 0),
				)
				return m
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireMetricEqual(t, tt.want, tt.got)
		})
	}
}

func TestRequireMetricsEqual(t *testing.T) {
	tests := []struct {
		name string
		got  []internal.Metric
		want []internal.Metric
		opts []cmp.Option
	}{
		{
			name: "sort metrics option sorts by name",
			got: []internal.Metric{
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
			},
			want: []internal.Metric{
				MustMetric(
					"net",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
				MustMetric(
					"cpu",
					map[string]string{},
					map[string]interface{}{},
					time.Unix(0, 0),
				),
			},
			opts: []cmp.Option{SortMetrics()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RequireMetricsEqual(t, tt.want, tt.got, tt.opts...)
		})
	}
}
