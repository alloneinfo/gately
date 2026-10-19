package module

import (
	"strings"
	"testing"
)

func TestWriteSamples(t *testing.T) {
	var b strings.Builder
	err := WriteSamples(&b, []Sample{
		{Name: "a_total", Help: "A counter", Type: MetricCounter, Labels: map[string]string{"z": "1", "a": `x"y`}, Value: 3},
		{Name: "b", Value: 1.5},
		{Name: "a_total", Labels: map[string]string{"a": "2"}, Value: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "# HELP a_total A counter\n# TYPE a_total counter\na_total{a=\"x\\\"y\",z=\"1\"} 3\na_total{a=\"2\"} 4\n# TYPE b gauge\nb 1.5\n"
	if b.String() != want {
		t.Fatalf("输出不符:\n%s\n--- want ---\n%s", b.String(), want)
	}
}
