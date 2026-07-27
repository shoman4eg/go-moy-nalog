package moynalog

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// timeLayouts lists every timestamp shape the API has been observed to emit.
// Responses mix full RFC 3339 timestamps with bare dates ("2022-11-12").
var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// Time wraps time.Time to decode the several timestamp formats the API mixes,
// and to encode requests in the format it expects (RFC 3339, second precision).
// A JSON null decodes to the zero Time and a zero Time encodes back to null.
type Time struct {
	time.Time
}

// NewTime returns a Time wrapping t.
func NewTime(t time.Time) Time {
	return Time{Time: t}
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Time) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(string(data), `"`)
	if raw == "" || raw == "null" {
		t.Time = time.Time{}

		return nil
	}

	for _, layout := range timeLayouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			t.Time = parsed

			return nil
		}
	}

	return errors.Errorf("moynalog: cannot parse %q as a timestamp", raw)
}

// MarshalJSON implements json.Marshaler.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}

	return []byte(strconv.Quote(t.Format(time.RFC3339))), nil
}

// String returns the timestamp in the format used on the wire.
func (t Time) String() string {
	if t.IsZero() {
		return ""
	}

	return t.Format(time.RFC3339)
}

// EncodeValues implements query.Encoder so Time works in query parameters. The
// API expects the extended RFC 3339 form there, with millisecond precision.
func (t Time) EncodeValues(key string, v *url.Values) error {
	if t.IsZero() {
		return nil
	}
	v.Set(key, t.Format("2006-01-02T15:04:05.000Z07:00"))

	return nil
}
