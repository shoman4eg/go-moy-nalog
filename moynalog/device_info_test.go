package moynalog

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestStdDeviceIDGeneratorEncoding(t *testing.T) {
	t.Parallel()

	// The generator must base64 the raw material, drop "+", "/" and "=", then
	// truncate and lowercase it.
	generator := NewDeviceIDGenerator(IDStrategyFunc(func() (string, error) {
		return "\xff\xff\xfe some raw material that is definitely long enough", nil
	}))

	got, err := generator.DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}

	if len(got) != defaultDeviceIDLength {
		t.Errorf("length = %d, want %d", len(got), defaultDeviceIDLength)
	}
	if strings.ContainsAny(got, "+/=") {
		t.Errorf("DeviceID = %q, want it free of +, / and =", got)
	}
	if got != strings.ToLower(got) {
		t.Errorf("DeviceID = %q, want it lowercased", got)
	}
}

func TestStdDeviceIDGeneratorCustomLength(t *testing.T) {
	t.Parallel()

	got, err := NewStaticDeviceIDGenerator("seed", WithIDLength(10)).DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}
	if len(got) != 10 {
		t.Errorf("length = %d, want 10", len(got))
	}
}

// Turning lowercasing off must change nothing but the case.
func TestStdDeviceIDGeneratorWithoutLowercase(t *testing.T) {
	t.Parallel()

	raw, err := NewStaticDeviceIDGenerator("seed", WithIDLowercase(false)).DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}
	lowercased, err := NewStaticDeviceIDGenerator("seed").DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}

	if strings.ToLower(raw) != lowercased {
		t.Errorf("lowercase(%q) = %q, want %q", raw, strings.ToLower(raw), lowercased)
	}
}

// The encoded identifier must only contain characters the API accepts.
func TestStdDeviceIDGeneratorAlphabet(t *testing.T) {
	t.Parallel()

	got, err := NewStaticDeviceIDGenerator("seed").DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}

	if len(got) != defaultDeviceIDLength {
		t.Errorf("length = %d, want %d", len(got), defaultDeviceIDLength)
	}
	if !regexp.MustCompile(`^[a-z0-9]+$`).MatchString(got) {
		t.Errorf("DeviceID = %q, want only lowercase letters and digits", got)
	}
}

// The same seed must always produce the same identifier.
func TestStaticDeviceIDGeneratorIsDeterministic(t *testing.T) {
	t.Parallel()

	first, err := NewStaticDeviceIDGenerator("seed").DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}
	second, err := NewStaticDeviceIDGenerator("seed").DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}
	if first != second {
		t.Errorf("static generator returned %q then %q", first, second)
	}

	other, err := NewStaticDeviceIDGenerator("another seed").DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}
	if other == first {
		t.Error("different seeds must produce different identifiers")
	}
}

func TestRandomDeviceIDGeneratorVaries(t *testing.T) {
	t.Parallel()

	generator := NewRandomDeviceIDGenerator()

	first, err := generator.DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}
	second, err := generator.DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}
	if first == second {
		t.Error("the random generator returned the same identifier twice")
	}
}

// The platform strategy is what ties receipts to a machine, so it must be
// stable across generator instances.
func TestPlatformDeviceIDGeneratorIsStable(t *testing.T) {
	t.Parallel()

	first, err := NewPlatformDeviceIDGenerator().DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}
	second, err := NewPlatformDeviceIDGenerator().DeviceID()
	if err != nil {
		t.Fatalf("DeviceID: %v", err)
	}
	if first != second {
		t.Errorf("platform generator returned %q then %q", first, second)
	}
	if first == "" {
		t.Error("platform generator returned an empty identifier")
	}
}

// A failing generator must surface through the authentication calls rather than
// be silently swallowed at construction.
func TestDeviceIDErrorSurfacesOnAuth(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("no device id here")
	client, _ := setup(t)
	client = NewClient(
		WithEndpoint(client.BaseURL.String()),
		WithDeviceIDGenerator(DeviceIDFunc(func() (string, error) { return "", sentinel })),
	)

	_, _, err := client.Auth.CreateAccessToken(context.Background(), "inn", "password")
	if !errors.Is(err, sentinel) {
		t.Errorf("error = %v, want the generator error", err)
	}
}

func TestNewDeviceInfoJSON(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(NewDeviceInfo("device-id", "agent"))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	want := `{"sourceType":"WEB","sourceDeviceId":"device-id","appVersion":"1.0.0","metaDetails":{"userAgent":"agent"}}`
	if string(got) != want {
		t.Errorf("DeviceInfo JSON =\n%s\nwant\n%s", got, want)
	}
}
