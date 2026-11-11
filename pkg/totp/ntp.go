package totp

import (
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultNTPTimeout is the timeout for NTP UDP requests
	DefaultNTPTimeout = 2 * time.Second

	// DefaultHTTPTimeout is the timeout for HTTP Date requests
	DefaultHTTPTimeout = 2 * time.Second

	// DriftThresholdSec is the threshold in seconds to consider clock drifted (> 5s)
	DriftThresholdSec = 5.0

	// ntpEpochOffset is the number of seconds between 1900-01-01 and 1970-01-01
	ntpEpochOffset = 2208988800
)

// DefaultNTPServers contains default public NTP servers
var DefaultNTPServers = []string{
	"pool.ntp.org:123",
	"time.google.com:123",
	"time.cloudflare.com:123",
	"ntp.aliyun.com:123",
}

// DefaultHTTPServers contains reliable HTTP endpoints for Date header fallback
var DefaultHTTPServers = []string{
	"https://www.cloudflare.com",
	"https://www.google.com",
	"https://www.aliyun.com",
}

// TimeDriftResult represents clock drift detection outcome
type TimeDriftResult struct {
	SystemTime   time.Time     `json:"system_time"`
	NetworkTime  time.Time     `json:"network_time"`
	Offset       time.Duration `json:"offset"`
	DriftSeconds float64       `json:"drift_seconds"`
	IsDrifted    bool          `json:"is_drifted"`
	Source       string        `json:"source"`
	Server       string        `json:"server,omitempty"`
}

// NTPClient performs time drift detection with customizable servers and timeouts
type NTPClient struct {
	NTPServers  []string
	HTTPServers []string
	Timeout     time.Duration
	HTTPClient  *http.Client
}

// NewNTPClient returns an NTPClient with default configurations
func NewNTPClient() *NTPClient {
	return &NTPClient{
		NTPServers:  DefaultNTPServers,
		HTTPServers: DefaultHTTPServers,
		Timeout:     DefaultNTPTimeout,
		HTTPClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
			},
		},
	}
}

// QueryNTP queries an NTP server over UDP and returns the server time and clock offset
func QueryNTP(server string, timeout time.Duration) (time.Time, time.Duration, error) {
	if !strings.Contains(server, ":") {
		server += ":123"
	}

	conn, err := net.DialTimeout("udp", server, timeout)
	if err != nil {
		return time.Time{}, 0, err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return time.Time{}, 0, err
	}

	// NTP packet is 48 bytes.
	// First byte: LI=0 (2 bits), VN=4 (3 bits), Mode=3 (client, 3 bits) -> 0x23
	req := make([]byte, 48)
	req[0] = 0x23

	t1 := time.Now()
	if _, err := conn.Write(req); err != nil {
		return time.Time{}, 0, err
	}

	resp := make([]byte, 48)
	n, err := conn.Read(resp)
	t4 := time.Now()
	if err != nil {
		return time.Time{}, 0, err
	}
	if n < 48 {
		return time.Time{}, 0, fmt.Errorf("invalid NTP response length: %d bytes", n)
	}

	mode := resp[0] & 0x07
	if mode != 4 && mode != 5 {
		return time.Time{}, 0, fmt.Errorf("invalid NTP response mode: %d", mode)
	}

	stratum := resp[1]
	if stratum == 0 {
		return time.Time{}, 0, errors.New("NTP server returned Kiss-o'-Death (stratum 0)")
	}

	// Transmit timestamp is bytes 40-47
	sec := binary.BigEndian.Uint32(resp[40:44])
	frac := binary.BigEndian.Uint32(resp[44:48])

	if sec == 0 {
		return time.Time{}, 0, errors.New("invalid zero NTP transmit timestamp")
	}

	unixSec := int64(sec) - ntpEpochOffset
	nanoSec := int64((uint64(frac) * 1_000_000_000) >> 32)
	serverTime := time.Unix(unixSec, nanoSec).UTC()

	// Offset = ((T2 - T1) + (T3 - T4)) / 2
	// Approximated as serverTime - midpoint(T1, T4)
	rtt := t4.Sub(t1)
	clientEstimate := t1.Add(rtt / 2)
	offset := serverTime.Sub(clientEstimate)

	return serverTime, offset, nil
}

// QueryHTTP queries an HTTP server's Date header and returns the server time and clock offset
func QueryHTTP(url string, client *http.Client, timeout time.Duration) (time.Time, time.Duration, error) {
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return time.Time{}, 0, err
	}
	req.Header.Set("User-Agent", "2fa-vault-ntp/1.0")

	t1 := time.Now()
	resp, err := client.Do(req)
	t4 := time.Now()
	if err != nil {
		return time.Time{}, 0, err
	}
	defer resp.Body.Close()

	dateHeader := resp.Header.Get("Date")
	if dateHeader == "" {
		return time.Time{}, 0, errors.New("no Date header found in HTTP response")
	}

	serverTime, err := http.ParseTime(dateHeader)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("failed to parse HTTP Date header %q: %w", dateHeader, err)
	}

	rtt := t4.Sub(t1)
	clientEstimate := t1.Add(rtt / 2)
	offset := serverTime.Sub(clientEstimate)

	return serverTime, offset, nil
}

// DetectTimeDrift checks clock drift using NTP first, falling back to HTTP Date
func (c *NTPClient) DetectTimeDrift() (*TimeDriftResult, error) {
	var lastErr error

	// 1. Try NTP servers first
	for _, srv := range c.NTPServers {
		serverTime, offset, err := QueryNTP(srv, c.Timeout)
		if err == nil {
			driftSec := offset.Seconds()
			driftSec = math.Round(driftSec*1000) / 1000
			now := time.Now()
			return &TimeDriftResult{
				SystemTime:   now,
				NetworkTime:  serverTime,
				Offset:       offset,
				DriftSeconds: driftSec,
				IsDrifted:    math.Abs(driftSec) > DriftThresholdSec,
				Source:       "ntp",
				Server:       srv,
			}, nil
		}
		lastErr = err
	}

	// 2. Fall back to HTTP servers if UDP is blocked by firewall
	for _, url := range c.HTTPServers {
		serverTime, offset, err := QueryHTTP(url, c.HTTPClient, c.Timeout)
		if err == nil {
			driftSec := offset.Seconds()
			driftSec = math.Round(driftSec*1000) / 1000
			now := time.Now()
			return &TimeDriftResult{
				SystemTime:   now,
				NetworkTime:  serverTime,
				Offset:       offset,
				DriftSeconds: driftSec,
				IsDrifted:    math.Abs(driftSec) > DriftThresholdSec,
				Source:       "http",
				Server:       url,
			}, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to detect time drift: %w", lastErr)
	}
	return nil, errors.New("no time servers configured")
}

// DetectTimeDrift performs default time drift detection
func DetectTimeDrift() (*TimeDriftResult, error) {
	return NewNTPClient().DetectTimeDrift()
}

// GenerateAt calculates the OTP code at a specific time
func GenerateAt(opts Options, t time.Time) (string, error) {
	return GenerateCode(opts, t)
}

// GenerateCodeWithOffset calculates OTP code applying a clock drift offset
func GenerateCodeWithOffset(opts Options, t time.Time, offset time.Duration) (string, error) {
	return GenerateCode(opts, t.Add(offset))
}

// GenerateNextCode generates the OTP code for the upcoming period cycle
func GenerateNextCode(opts Options, t time.Time) (string, error) {
	if opts.Type == TypeHOTP {
		opts.Counter++
		return GenerateCode(opts, t)
	}
	period := opts.Period
	if period <= 0 {
		period = 30
	}
	nextTime := t.Add(time.Duration(period) * time.Second)
	return GenerateCode(opts, nextTime)
}

// GenerateNextCodeWithOffset generates the next cycle OTP code with clock offset
func GenerateNextCodeWithOffset(opts Options, t time.Time, offset time.Duration) (string, error) {
	return GenerateNextCode(opts, t.Add(offset))
}
