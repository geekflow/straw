package http

import (
	"fmt"
	"geeksaga.com/os/straw/internal"
	"geeksaga.com/os/straw/metric"
	"geeksaga.com/os/straw/plugins/serializer/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func getMetric() internal.Metric {
	m, err := metric.New(
		"cpu",
		map[string]string{},
		map[string]interface{}{
			"value": 42.0,
		},
		time.Unix(0, 0),
	)
	if err != nil {
		panic(err)
	}
	return m
}

func TestInvalidMethod(t *testing.T) {
	plugin := &HTTP{
		URL:    "",
		Method: http.MethodGet,
	}

	err := plugin.Connect()
	require.Error(t, err)
}

func TestMethod(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	defer ts.Close()

	u, err := url.Parse(fmt.Sprintf("http://%s", ts.Listener.Addr().String()))
	require.NoError(t, err)

	tests := []struct {
		name           string
		plugin         *HTTP
		expectedMethod string
		connectError   bool
	}{
		{
			name: "default method is POST",
			plugin: &HTTP{
				URL:    u.String(),
				Method: defaultMethod,
			},
			expectedMethod: http.MethodPost,
		},
		{
			name: "put is okay",
			plugin: &HTTP{
				URL:    u.String(),
				Method: http.MethodPut,
			},
			expectedMethod: http.MethodPut,
		},
		{
			name: "get is invalid",
			plugin: &HTTP{
				URL:    u.String(),
				Method: http.MethodGet,
			},
			connectError: true,
		},
		{
			name: "method is case insensitive",
			plugin: &HTTP{
				URL:    u.String(),
				Method: "poST",
			},
			expectedMethod: http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, tt.expectedMethod, r.Method)
				w.WriteHeader(http.StatusOK)
			})

			//serializer := influx.NewSerializer()
			//tt.plugin.SetSerializer(serializer)
			err = tt.plugin.Connect()
			if tt.connectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			err = tt.plugin.Write([]internal.Metric{getMetric()})
			require.NoError(t, err)
		})
	}
}

func TestDefaultUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	defer ts.Close()

	u, err := url.Parse(fmt.Sprintf("http://%s", ts.Listener.Addr().String()))
	require.NoError(t, err)

	internal.SetVersion("0.0.1")

	t.Run("default-user-agent", func(t *testing.T) {
		ts.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "Straw/0.0.1", r.Header.Get("User-Agent"))
			w.WriteHeader(http.StatusOK)
		})

		client := &HTTP{
			URL:    u.String(),
			Method: defaultMethod,
		}

		//serializer := influx.NewSerializer()
		//client.SetSerializer(serializer)

		serializer, _ := json.NewSerializer(65 * time.Millisecond)
		client.SetSerializer(serializer)

		err = client.Connect()
		require.NoError(t, err)

		err = client.Write([]internal.Metric{getMetric()})
		require.NoError(t, err)
	})
}
