package perfdata

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/NETWAYS/go-check"
)

// Replace not allowed characters inside a label
var replacer = strings.NewReplacer("=", "_", "`", "_", "'", "_", "\"", "_")

// formatNumeric returns a string representation of various possible numerics
//
// This supports most internal types of Go and all fmt.Stringer interfaces.
// Returns an eror in some known cases where the value of a data type does not
// represent a valid measurement, e.g INF for floats
// This error can probably ignored in most cases and the perfdata point omitted,
// but silently dropping the value and returning the empty strings seems like bad style
func formatNumeric(value PerfdataValue) (string, error) {
	switch value.kind {
	case floatType:
		if math.IsInf(value.floatVal, 0) {
			return "", errors.New("Perfdata value is inifinite")
		}

		if math.IsNaN(value.floatVal) {
			return "", errors.New("Perfdata value is inifinite")
		}

		return check.FormatFloat(value.floatVal), nil
	case intType:
		return fmt.Sprintf("%d", value.intVal), nil
	case uintType:
		return fmt.Sprintf("%d", value.uintVal), nil
	default:
		return "", errors.New("This should not happen")
	}
}

// Perfdata represents all properties of performance data for Icinga
//
// Implements fmt.Stringer to return the plaintext format for a plugin output.
//
// For examples of Uom see:
//
// https://www.monitoring-plugins.org/doc/guidelines.html#AEN201
//
// https://github.com/Icinga/icinga2/blob/master/lib/base/perfdatavalue.cpp
//
// https://icinga.com/docs/icinga-2/latest/doc/05-service-monitoring/#unit-of-measurement-uom
type Perfdata struct {
	Label string
	Value PerfdataValue
	// Uom is the unit-of-measurement, see links above for details.
	Uom  string
	Warn *check.Threshold
	Crit *check.Threshold
	Min  PerfdataValue
	Max  PerfdataValue
}

type perfdataValueTypeEnum int

const (
	noneType perfdataValueTypeEnum = iota
	intType
	uintType
	floatType
)

type PerfdataValue struct {
	kind     perfdataValueTypeEnum
	uintVal  uint64
	intVal   int64
	floatVal float64
}

func NewPdvUint64(val uint64) PerfdataValue {
	return PerfdataValue{
		kind:    uintType,
		uintVal: val,
	}
}

func NewPdvInt64(val int64) PerfdataValue {
	return PerfdataValue{
		kind:   intType,
		intVal: val,
	}
}

func NewPdvFloat64(val float64) PerfdataValue {
	return PerfdataValue{
		kind:     floatType,
		floatVal: val,
	}
}

// String returns the proper format for the plugin output
// on errors (occurs with invalid data, the empty string is returned
func (p Perfdata) String() string {
	tmp, _ := p.ValidatedString()
	return tmp
}

// ValidatedString returns the proper format for the plugin output
// Returns an eror in some known cases where the value of a data type does not
// represent a valid measurement, see the explanation for "formatNumeric" for
// perfdata values.
func (p Perfdata) ValidatedString() (string, error) {
	var sb strings.Builder

	// Add quotes if string contains any whitespace
	if strings.ContainsAny(p.Label, "\t\n\f\r ") {
		sb.WriteString(`'` + replacer.Replace(p.Label) + `'` + "=")
	} else {
		sb.WriteString(replacer.Replace(p.Label) + "=")
	}

	pfVal, err := formatNumeric(p.Value)
	if err != nil {
		return "", err
	}

	sb.WriteString(pfVal)
	sb.WriteString(p.Uom)

	// Thresholds
	for _, value := range []*check.Threshold{p.Warn, p.Crit} {
		sb.WriteString(";")

		if value != nil {
			sb.WriteString(value.String())
		}
	}

	// Limits
	for _, value := range []PerfdataValue{p.Min, p.Max} {
		sb.WriteString(";")

		if value.kind != noneType {
			pfVal, err := formatNumeric(value)
			// Attention: we ignore limits if they are faulty
			if err == nil {
				sb.WriteString(pfVal)
			}
		}
	}

	return strings.TrimRight(sb.String(), ";"), nil
}
