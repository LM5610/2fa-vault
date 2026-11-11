package totp

import (
	"encoding/binary"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryNTP_MockServer(t *testing.T) {
	// Start mock NTP UDP server
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen UDP: %v", err)
	}
	defer pc.Close()

	// Desired mock server time: 10 seconds ahead of client
	mockTime := time.Now().Add(10 * time.Second).UTC()

	go func() {
		buf := make([]byte, 48)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			return
		}
		if n < 48 {
			return
		}

		resp := make([]byte, 48)
		// Mode 4 = server, VN = 4
		resp[0] = 0x24
		resp[1] = 1 // Stratum 1

		unixSec := mockTime.Unix()
		sec := uint32(unixSec + ntpEpochOffset)
		frac := uint32((uint64(mockTime.Nanosecond()) << 32) / 1_000_000_000)

		binary.BigEndian.PutUint32(resp[40:44], sec)
		binary.BigEndian.PutUint32(resp[44:48], frac)

		_, _ = pc.WriteTo(resp, addr)
	}()

	serverTime, offset, err := QueryNTP(pc.LocalAddr().String(), 2*time.Second)
	if err != nil {
		t.Fatalf("QueryNTP failed: %v", err)
	}

	if math.Abs(offset.Seconds()-10.0) > 1.0 {
		t.Errorf("expected offset ~10s, got %v", offset)
	}
	if math.Abs(serverTime.Sub(mockTime).Seconds()) > 1.0 {
		t.Errorf("expected serverTime ~%v, got %v", mockTime, serverTime)
	}
}

func TestQueryHTTP_MockServer(t *testing.T) {
	mockTime := time.Now().Add(8 * time.Second).UTC()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", mockTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	serverTime, offset, err := QueryHTTP(ts.URL, ts.Client(), 2*time.Second)
	if err != nil {
		t.Fatalf("QueryHTTP failed: %v", err)
	}

	if math.Abs(offset.Seconds()-8.0) > 1.5 {
		t.Errorf("expected offset ~8s, got %v", offset)
	}
	if math.Abs(serverTime.Sub(mockTime).Seconds()) > 1.5 {
		t.Errorf("expected serverTime ~%v, got %v", mockTime, serverTime)
	}
}

func TestDetectTimeDrift_FallbackToHTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", time.Now().Add(10*time.Second).UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &NTPClient{
		NTPServers:  []string{"127.0.0.1:1"}, // Unreachable UDP port triggers fast failure
		HTTPServers: []string{ts.URL},
		Timeout:     100 * time.Millisecond,
		HTTPClient:  ts.Client(),
	}

	res, err := client.DetectTimeDrift()
	if err != nil {
		t.Fatalf("DetectTimeDrift fallback failed: %v", err)
	}

	if res.Source != "http" {
		t.Errorf("expected source 'http', got %s", res.Source)
	}
	if !res.IsDrifted {
		t.Errorf("expected is_drifted true for ~10s offset, got false")
	}
	if math.Abs(res.DriftSeconds-10.0) > 2.0 {
		t.Errorf("expected drift_seconds ~10.0, got %f", res.DriftSeconds)
	}
}

func TestDetectTimeDrift_NoDrift(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", time.Now().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := &NTPClient{
		NTPServers:  []string{"127.0.0.1:1"},
		HTTPServers: []string{ts.URL},
		Timeout:     100 * time.Millisecond,
		HTTPClient:  ts.Client(),
	}

	res, err := client.DetectTimeDrift()
	if err != nil {
		t.Fatalf("DetectTimeDrift failed: %v", err)
	}

	if res.IsDrifted {
		t.Errorf("expected is_drifted false for synced clock, got true")
	}
}

func TestTOTP_GenerateAt_And_GenerateNextCode(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP" // standard base32 test secret
	opts := Options{
		Secret:    secret,
		Digits:    6,
		Period:    30,
		Algorithm: AlgorithmSHA1,
		Type:      TypeTOTP,
	}

	now := time.Unix(1111111110, 0) // counter = 37037037
	code, err := GenerateCode(opts, now)
	if err != nil {
		t.Fatalf("GenerateCode failed: %v", err)
	}

	codeAt, err := GenerateAt(opts, now)
	if err != nil {
		t.Fatalf("GenerateAt failed: %v", err)
	}
	if code != codeAt {
		t.Errorf("expected GenerateAt to match GenerateCode: %s vs %s", code, codeAt)
	}

	// GenerateNextCode should generate code for now + 30s
	nextCode, err := GenerateNextCode(opts, now)
	if err != nil {
		t.Fatalf("GenerateNextCode failed: %v", err)
	}
	expectedNext, err := GenerateCode(opts, now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("GenerateCode for next cycle failed: %v", err)
	}
	if nextCode != expectedNext {
		t.Errorf("expected NextCode %s, got %s", expectedNext, nextCode)
	}
	if nextCode == code {
		t.Errorf("NextCode should differ from current code, got both %s", code)
	}

	// Test offset
	offset := 30 * time.Second
	offsetCode, err := GenerateCodeWithOffset(opts, now, offset)
	if err != nil {
		t.Fatalf("GenerateCodeWithOffset failed: %v", err)
	}
	if offsetCode != nextCode {
		t.Errorf("offset of 30s should produce nextCode %s, got %s", nextCode, offsetCode)
	}

	// Test NextCode with offset
	nextOffsetCode, err := GenerateNextCodeWithOffset(opts, now, offset)
	if err != nil {
		t.Fatalf("GenerateNextCodeWithOffset failed: %v", err)
	}
	expectedAfterNext, err := GenerateCode(opts, now.Add(60*time.Second))
	if err != nil {
		t.Fatalf("GenerateCode for +60s failed: %v", err)
	}
	if nextOffsetCode != expectedAfterNext {
		t.Errorf("expected %s, got %s", expectedAfterNext, nextOffsetCode)
	}
}

func TestHOTP_GenerateNextCode(t *testing.T) {
	opts := Options{
		Secret:    "JBSWY3DPEHPK3PXP",
		Digits:    6,
		Algorithm: AlgorithmSHA1,
		Type:      TypeHOTP,
		Counter:   1,
	}

	now := time.Now()
	nextCode, err := GenerateNextCode(opts, now)
	if err != nil {
		t.Fatalf("HOTP GenerateNextCode failed: %v", err)
	}

	optsCounter2 := opts
	optsCounter2.Counter = 2
	expected, _ := GenerateCode(optsCounter2, now)
	if nextCode != expected {
		t.Errorf("HOTP NextCode expected %s, got %s", expected, nextCode)
	}
}
