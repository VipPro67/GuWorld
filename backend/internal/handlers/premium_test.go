package handlers

import (
	"fmt"
	"os"
	"testing"
)

func TestPayOSDescriptionLength(t *testing.T) {
	testOrderCodes := []int64{
		100000000000,
		999999999999,
		123456789012,
		888888888888,
	}

	for _, orderCode := range testOrderCodes {
		desc := fmt.Sprintf("GU%06d", orderCode%1000000)
		if len(desc) > 9 {
			t.Errorf("description %q length %d exceeds PayOS limit of 9 chars for orderCode %d", desc, len(desc), orderCode)
		}
	}
}

func TestVerifyPayOSSignature(t *testing.T) {
	checksumKey := "test_checksum_key_123"
	os.Setenv("PAYOS_CHECKSUM_KEY", checksumKey)
	defer os.Unsetenv("PAYOS_CHECKSUM_KEY")

	data := map[string]interface{}{
		"orderCode":           float64(123456789),
		"amount":              float64(50000),
		"description":         "GU123456",
		"accountNumber":       "1234567890",
		"reference":           "REF123",
		"transactionDateTime": "2026-09-03 20:00:00",
		"currency":            "VND",
		"paymentLinkId":       "link123",
		"code":                "00",
		"desc":                "success",
	}

	sig1 := verifyPayOSSignature(data, "invalid_sig")
	if sig1 != false {
		t.Errorf("expected false for invalid signature")
	}

	dataWithSignature := make(map[string]interface{})
	for k, v := range data {
		dataWithSignature[k] = v
	}
	dataWithSignature["signature"] = "some_signature_value"

	sigPayload := "accountNumber=1234567890&amount=50000&code=00&currency=VND&desc=success&description=GU123456&orderCode=123456789&paymentLinkId=link123&reference=REF123&transactionDateTime=2026-09-03 20:00:00"
	expectedSig := hmacSHA256Hex(sigPayload, checksumKey)

	if !verifyPayOSSignature(dataWithSignature, expectedSig) {
		t.Errorf("verifyPayOSSignature failed for data containing 'signature' key")
	}
}

func TestPayOSValueString(t *testing.T) {
	if s := payOSValueString(float64(10000)); s != "10000" {
		t.Errorf("expected '10000', got %q", s)
	}
	if s := payOSValueString(float64(10000.5)); s != "10000.5" {
		t.Errorf("expected '10000.5', got %q", s)
	}
	if s := payOSValueString("hello"); s != "hello" {
		t.Errorf("expected 'hello', got %q", s)
	}
	if s := payOSValueString("undefined"); s != "" {
		t.Errorf("expected '', got %q", s)
	}
	if s := payOSValueString("null"); s != "" {
		t.Errorf("expected '', got %q", s)
	}
	if s := payOSValueString(true); s != "true" {
		t.Errorf("expected 'true', got %q", s)
	}
}
