package moynalog

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"runtime"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/pkg/errors"
)

const (
	// SourceTypeWeb is the only source type the web API accepts.
	SourceTypeWeb = "WEB"
	// AppVersion is reported to the API alongside the device identifier.
	AppVersion = "1.0.0"

	// defaultDeviceIDLength matches the identifier length the web client sends.
	defaultDeviceIDLength = 21
	// machineIDAppKey namespaces the derived machine identifier.
	machineIDAppKey = "go-moy-nalog"
	// randomIDBytes is how many random bytes RandomIDStrategy draws by default.
	randomIDBytes = 16
)

// DeviceInfo identifies the client to the API. Both authentication endpoints
// require it.
type DeviceInfo struct {
	SourceType     string `json:"sourceType"`
	SourceDeviceID string `json:"sourceDeviceId"`
	AppVersion     string `json:"appVersion"`
	MetaDetails    struct {
		UserAgent string `json:"userAgent"`
	} `json:"metaDetails"`
}

// NewDeviceInfo builds the device descriptor sent with authentication requests.
func NewDeviceInfo(deviceID, userAgent string) *DeviceInfo {
	di := &DeviceInfo{
		SourceType:     SourceTypeWeb,
		SourceDeviceID: deviceID,
		AppVersion:     AppVersion,
	}
	di.MetaDetails.UserAgent = userAgent

	return di
}

// IDStrategy produces the raw material a device identifier is derived from.
// Implement it to control how a host is identified to the API.
type IDStrategy interface {
	// ID returns the raw identifier material.
	ID() (string, error)
}

// IDStrategyFunc adapts a plain function to IDStrategy.
type IDStrategyFunc func() (string, error)

// ID implements IDStrategy.
func (f IDStrategyFunc) ID() (string, error) { return f() }

// DeviceIDGenerator produces the device identifier reported to the API. The
// API ties receipts to the device that registered them, so a generator should
// normally return the same value across runs on the same host.
type DeviceIDGenerator interface {
	// DeviceID returns the identifier to report.
	DeviceID() (string, error)
}

// DeviceIDFunc adapts a plain function to DeviceIDGenerator.
type DeviceIDFunc func() (string, error)

// DeviceID implements DeviceIDGenerator.
func (f DeviceIDFunc) DeviceID() (string, error) { return f() }

// StdDeviceIDGenerator derives an identifier by base64 encoding the output of
// its strategy, dropping the non-alphanumeric characters of the alphabet and
// truncating the result. It mirrors the reference PHP implementation.
type StdDeviceIDGenerator struct {
	// Strategy supplies the raw identifier material. Defaults to
	// PlatformIDStrategy when nil.
	Strategy IDStrategy
	// Length truncates the encoded identifier. Defaults to 21 when zero.
	Length int
	// Lowercase lowercases the encoded identifier.
	Lowercase bool
}

// GeneratorOption customises a StdDeviceIDGenerator.
type GeneratorOption func(*StdDeviceIDGenerator)

// WithIDLength sets the length the encoded identifier is truncated to.
func WithIDLength(length int) GeneratorOption {
	return func(g *StdDeviceIDGenerator) {
		if length > 0 {
			g.Length = length
		}
	}
}

// WithIDLowercase controls whether the encoded identifier is lowercased.
func WithIDLowercase(lowercase bool) GeneratorOption {
	return func(g *StdDeviceIDGenerator) {
		g.Lowercase = lowercase
	}
}

// NewDeviceIDGenerator returns a generator that encodes the output of strategy.
func NewDeviceIDGenerator(strategy IDStrategy, opts ...GeneratorOption) *StdDeviceIDGenerator {
	g := &StdDeviceIDGenerator{
		Strategy:  strategy,
		Length:    defaultDeviceIDLength,
		Lowercase: true,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// NewPlatformDeviceIDGenerator returns the default generator, which derives a
// stable identifier from the host machine.
func NewPlatformDeviceIDGenerator(opts ...GeneratorOption) *StdDeviceIDGenerator {
	return NewDeviceIDGenerator(PlatformIDStrategy{}, opts...)
}

// NewRandomDeviceIDGenerator returns a generator that draws a fresh identifier
// on every call. Useful for tests; a new identifier per run makes the API treat
// each run as a different device.
func NewRandomDeviceIDGenerator(opts ...GeneratorOption) *StdDeviceIDGenerator {
	return NewDeviceIDGenerator(RandomIDStrategy{}, opts...)
}

// NewStaticDeviceIDGenerator returns a generator that always derives its
// identifier from value.
func NewStaticDeviceIDGenerator(value string, opts ...GeneratorOption) *StdDeviceIDGenerator {
	return NewDeviceIDGenerator(StaticIDStrategy{Value: value}, opts...)
}

// DeviceID implements DeviceIDGenerator.
func (g *StdDeviceIDGenerator) DeviceID() (string, error) {
	strategy := g.Strategy
	if strategy == nil {
		strategy = PlatformIDStrategy{}
	}

	raw, err := strategy.ID()
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(raw))
	encoded = strings.NewReplacer("+", "", "/", "", "=", "").Replace(encoded)

	length := g.Length
	if length <= 0 {
		length = defaultDeviceIDLength
	}
	if len(encoded) > length {
		encoded = encoded[:length]
	}

	if g.Lowercase {
		encoded = strings.ToLower(encoded)
	}

	return encoded, nil
}

// PlatformIDStrategy derives identifier material from the host machine, so the
// same host keeps reporting the same device across runs.
type PlatformIDStrategy struct{}

// ID implements IDStrategy.
func (PlatformIDStrategy) ID() (string, error) {
	if id, err := machineid.ProtectedID(machineIDAppKey); err == nil && id != "" {
		return id, nil
	}

	// machineid has no source of truth on some hosts (containers, restricted
	// filesystems). Fall back to whatever identifies this host.
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.Wrap(err, "moynalog: cannot derive a platform device id")
	}

	return strings.Join([]string{hostname, runtime.GOOS, runtime.GOARCH, runtime.Version()}, "-"), nil
}

// RandomIDStrategy draws identifier material from a cryptographic source.
type RandomIDStrategy struct {
	// Length is how many random bytes to draw. Defaults to 16 when zero.
	Length int
}

// ID implements IDStrategy.
func (s RandomIDStrategy) ID() (string, error) {
	length := s.Length
	if length <= 0 {
		length = randomIDBytes
	}

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.Wrap(err, "moynalog: cannot read random bytes")
	}

	return string(buf), nil
}

// StaticIDStrategy derives identifier material by hashing a caller supplied
// value, which keeps the device identifier stable and reproducible.
type StaticIDStrategy struct {
	// Value is hashed to produce the identifier material.
	Value string
}

// ID implements IDStrategy.
func (s StaticIDStrategy) ID() (string, error) {
	sum := sha256.Sum256([]byte(s.Value))

	return string(sum[:]), nil
}
