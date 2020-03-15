package metric

import (
	"fmt"
	"geeksaga.com/os/straw/internal"
	"sort"
	"time"
)

type metric struct {
	name   string
	tags   []*internal.Tag
	fields []*internal.Field
	tm     time.Time

	tp        internal.ValueType
	aggregate bool
}

func New(
	name string,
	tags map[string]string,
	fields map[string]interface{},
	tm time.Time,
	tp ...internal.ValueType,
) (internal.Metric, error) {
	var valueType internal.ValueType
	if len(tp) > 0 {
		valueType = tp[0]
	} else {
		valueType = internal.Untyped
	}

	m := &metric{
		name:   name,
		tags:   nil,
		fields: nil,
		tm:     tm,
		tp:     valueType,
	}

	if len(tags) > 0 {
		m.tags = make([]*internal.Tag, 0, len(tags))
		for k, v := range tags {
			m.tags = append(m.tags,
				&internal.Tag{Key: k, Value: v})
		}
		sort.Slice(m.tags, func(i, j int) bool { return m.tags[i].Key < m.tags[j].Key })
	}

	m.fields = make([]*internal.Field, 0, len(fields))
	for k, v := range fields {
		v := convertField(v)
		if v == nil {
			continue
		}
		m.AddField(k, v)
	}

	return m, nil
}

// FromMetric returns a deep copy of the metric with any tracking information
// removed.
func FromMetric(other internal.Metric) internal.Metric {
	m := &metric{
		name:   other.Name(),
		tags:   make([]*internal.Tag, len(other.TagList())),
		fields: make([]*internal.Field, len(other.FieldList())),
		tm:     other.Time(),
		tp:     other.Type(),
		//aggregate: other.IsAggregate(),
	}

	for i, tag := range other.TagList() {
		m.tags[i] = &internal.Tag{Key: tag.Key, Value: tag.Value}
	}

	for i, field := range other.FieldList() {
		m.fields[i] = &internal.Field{Key: field.Key, Value: field.Value}
	}
	return m
}

func (m *metric) String() string {
	return fmt.Sprintf("%s %v %v %d", m.name, m.Tags(), m.Fields(), m.tm.UnixNano())
}

func (m *metric) Name() string {
	return m.name
}

func (m *metric) Tags() map[string]string {
	tags := make(map[string]string, len(m.tags))
	for _, tag := range m.tags {
		tags[tag.Key] = tag.Value
	}
	return tags
}

func (m *metric) TagList() []*internal.Tag {
	return m.tags
}

func (m *metric) Fields() map[string]interface{} {
	fields := make(map[string]interface{}, len(m.fields))
	for _, field := range m.fields {
		fields[field.Key] = field.Value
	}

	return fields
}

func (m *metric) FieldList() []*internal.Field {
	return m.fields
}

func (m *metric) Time() time.Time {
	return m.tm
}

func (m *metric) Type() internal.ValueType {
	return m.tp
}

func (m *metric) SetName(name string) {
	m.name = name
}

func (m *metric) AddPrefix(prefix string) {
	m.name = prefix + m.name
}

func (m *metric) AddSuffix(suffix string) {
	m.name = m.name + suffix
}

func (m *metric) AddField(key string, value interface{}) {
	for i, field := range m.fields {
		if key == field.Key {
			m.fields[i] = &internal.Field{Key: key, Value: convertField(value)}
			return
		}
	}
	m.fields = append(m.fields, &internal.Field{Key: key, Value: convertField(value)})
}

func (m *metric) Copy() internal.Metric {
	m2 := &metric{
		name:      m.name,
		tags:      make([]*internal.Tag, len(m.tags)),
		fields:    make([]*internal.Field, len(m.fields)),
		tm:        m.tm,
		tp:        m.tp,
		aggregate: m.aggregate,
	}

	for i, tag := range m.tags {
		m2.tags[i] = &internal.Tag{Key: tag.Key, Value: tag.Value}
	}

	for i, field := range m.fields {
		m2.fields[i] = &internal.Field{Key: field.Key, Value: field.Value}
	}
	return m2
}

func (m *metric) Accept() {
}

func (m *metric) Reject() {
}

func (m *metric) Drop() {
}

// Convert field to a supported type or nil if unconvertible
func convertField(v interface{}) interface{} {
	switch v := v.(type) {
	case float64:
		return v
	case int64:
		return v
	case string:
		return v
	case bool:
		return v
	case int:
		return int64(v)
	case uint:
		return uint64(v)
	case uint64:
		return uint64(v)
	case []byte:
		return string(v)
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)
	case uint32:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint8:
		return uint64(v)
	case float32:
		return float64(v)
	case *float64:
		if v != nil {
			return *v
		}
	case *int64:
		if v != nil {
			return *v
		}
	case *string:
		if v != nil {
			return *v
		}
	case *bool:
		if v != nil {
			return *v
		}
	case *int:
		if v != nil {
			return int64(*v)
		}
	case *uint:
		if v != nil {
			return uint64(*v)
		}
	case *uint64:
		if v != nil {
			return uint64(*v)
		}
	case *[]byte:
		if v != nil {
			return string(*v)
		}
	case *int32:
		if v != nil {
			return int64(*v)
		}
	case *int16:
		if v != nil {
			return int64(*v)
		}
	case *int8:
		if v != nil {
			return int64(*v)
		}
	case *uint32:
		if v != nil {
			return uint64(*v)
		}
	case *uint16:
		if v != nil {
			return uint64(*v)
		}
	case *uint8:
		if v != nil {
			return uint64(*v)
		}
	case *float32:
		if v != nil {
			return float64(*v)
		}
	default:
		return nil
	}
	return nil
}
