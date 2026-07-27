package moynalog

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"
)

func TestTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want time.Time
	}{
		{
			name: "RFC 3339 with fractional seconds",
			raw:  `"2022-02-01T00:47:30.446Z"`,
			want: time.Date(2022, time.February, 1, 0, 47, 30, 446000000, time.UTC),
		},
		{
			name: "RFC 3339 with an offset",
			raw:  `"2022-03-30T22:46:06+02:00"`,
			want: time.Date(2022, time.March, 30, 22, 46, 6, 0, time.FixedZone("", 2*60*60)),
		},
		{
			name: "bare date",
			raw:  `"2022-11-12"`,
			want: time.Date(2022, time.November, 12, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "timestamp without a zone",
			raw:  `"2021-01-27T22:38:30.057957"`,
			want: time.Date(2021, time.January, 27, 22, 38, 30, 57957000, time.UTC),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got Time
			if err := json.Unmarshal([]byte(tt.raw), &got); err != nil {
				t.Fatalf("Unmarshal(%s): %v", tt.raw, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("Unmarshal(%s) = %s, want %s", tt.raw, got, tt.want)
			}
		})
	}
}

func TestTimeUnmarshalNull(t *testing.T) {
	t.Parallel()

	var got Time
	if err := json.Unmarshal([]byte(`null`), &got); err != nil {
		t.Fatalf("Unmarshal(null): %v", err)
	}
	if !got.IsZero() {
		t.Errorf("null decoded to %s, want the zero time", got)
	}
}

func TestTimeUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var got Time
	if err := json.Unmarshal([]byte(`"not a timestamp"`), &got); err == nil {
		t.Error("an unparsable timestamp must be reported")
	}
}

func TestTimeMarshalJSON(t *testing.T) {
	t.Parallel()

	value := NewTime(time.Date(2022, time.April, 1, 16, 0, 30, 0, time.FixedZone("MSK", 3*60*60)))

	got, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if want := `"2022-04-01T16:00:30+03:00"`; string(got) != want {
		t.Errorf("Marshal = %s, want %s", got, want)
	}
}

func TestTimeMarshalZeroIsNull(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(Time{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(got) != "null" {
		t.Errorf("Marshal = %s, want null", got)
	}
}

func TestTimeEncodeValues(t *testing.T) {
	t.Parallel()

	values := url.Values{}
	value := NewTime(time.Date(2021, time.March, 30, 22, 39, 54, 391000000, time.UTC))

	if err := value.EncodeValues("from", &values); err != nil {
		t.Fatalf("EncodeValues: %v", err)
	}
	if want := "2021-03-30T22:39:54.391Z"; values.Get("from") != want {
		t.Errorf("from = %q, want %q", values.Get("from"), want)
	}
}

func TestTimeEncodeValuesSkipsZero(t *testing.T) {
	t.Parallel()

	values := url.Values{}
	if err := (Time{}).EncodeValues("from", &values); err != nil {
		t.Fatalf("EncodeValues: %v", err)
	}
	if _, ok := values["from"]; ok {
		t.Error("a zero time must not be encoded into the query")
	}
}
