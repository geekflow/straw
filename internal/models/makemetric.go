package models

import (
	"geeksaga.com/os/straw/internal"
)

// Makemetric applies new metric plugin and agent measurement and tag
// settings.
func makemetric(
	metric internal.Metric,
	nameOverride string,
	namePrefix string,
	nameSuffix string,
	tags map[string]string,
	globalTags map[string]string,
) internal.Metric {
	if len(nameOverride) != 0 {
		metric.SetName(nameOverride)
	}

	if len(namePrefix) != 0 {
		metric.AddPrefix(namePrefix)
	}
	if len(nameSuffix) != 0 {
		metric.AddSuffix(nameSuffix)
	}

	return metric
}
