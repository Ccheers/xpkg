package kafka

import (
	"context"
	"testing"
	"time"
)

var kafkaStorageEndpoints = []string{}

func TestKafkaStorage_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kafka integration test in short mode")
		return
	}

	storage, err := NewStorage(kafkaStorageEndpoints)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.(*Storage).Close()

	ctx := context.Background()
	key := "test_key"
	value := "test_value"
	ttl := 10 * time.Second

	err = storage.SetEx(ctx, key, value, ttl)
	if err != nil {
		t.Fatalf("failed to set key: %v", err)
	}

	keys, err := storage.Keys(ctx, "test_")
	if err != nil {
		t.Fatalf("failed to get keys: %v", err)
	}

	found := false
	for _, k := range keys {
		if k == key {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to find key %s in keys %v", key, keys)
	}

	err = storage.Del(ctx, key)
	if err != nil {
		t.Fatalf("failed to delete key: %v", err)
	}

	time.Sleep(2 * time.Second)

	keys, err = storage.Keys(ctx, "test_")
	if err != nil {
		t.Fatalf("failed to get keys after deletion: %v", err)
	}

	for _, k := range keys {
		if k == key {
			t.Errorf("key %s should have been deleted but still found in keys %v", key, keys)
		}
	}
}

func TestKafkaStorage_TTL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kafka integration test in short mode")
		return
	}

	storage, err := NewStorage(kafkaStorageEndpoints)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.(*Storage).Close()

	ctx := context.Background()
	key := "test_ttl_key"
	value := "test_ttl_value"
	ttl := 2 * time.Second

	err = storage.SetEx(ctx, key, value, ttl)
	if err != nil {
		t.Fatalf("failed to set key with TTL: %v", err)
	}

	time.Sleep(1 * time.Second)

	keys, err := storage.Keys(ctx, "test_ttl_")
	if err != nil {
		t.Fatalf("failed to get keys before expiration: %v", err)
	}

	found := false
	for _, k := range keys {
		if k == key {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected to find key %s before expiration in keys %v", key, keys)
	}

	time.Sleep(3 * time.Second)

	keys, err = storage.Keys(ctx, "test_ttl_")
	if err != nil {
		t.Fatalf("failed to get keys after expiration: %v", err)
	}

	for _, k := range keys {
		if k == key {
			t.Errorf("key %s should have expired but still found in keys %v", key, keys)
		}
	}
}

func TestKafkaStorage_MultipleKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kafka integration test in short mode")
		return
	}

	storage, err := NewStorage(kafkaStorageEndpoints)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.(*Storage).Close()

	ctx := context.Background()
	prefix := "multi_test_"
	keys := []string{
		prefix + "key1",
		prefix + "key2",
		prefix + "key3",
	}
	value := "test_value"
	ttl := 30 * time.Second

	for _, key := range keys {
		err = storage.SetEx(ctx, key, value, ttl)
		if err != nil {
			t.Fatalf("failed to set key %s: %v", key, err)
		}
	}

	time.Sleep(2 * time.Second)

	foundKeys, err := storage.Keys(ctx, prefix)
	if err != nil {
		t.Fatalf("failed to get keys: %v", err)
	}

	if len(foundKeys) != len(keys) {
		t.Errorf("expected %d keys, got %d: %v", len(keys), len(foundKeys), foundKeys)
	}

	for _, expectedKey := range keys {
		found := false
		for _, foundKey := range foundKeys {
			if foundKey == expectedKey {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected key %s not found in results %v", expectedKey, foundKeys)
		}
	}

	err = storage.Del(ctx, keys[1])
	if err != nil {
		t.Fatalf("failed to delete key %s: %v", keys[1], err)
	}

	time.Sleep(2 * time.Second)

	foundKeys, err = storage.Keys(ctx, prefix)
	if err != nil {
		t.Fatalf("failed to get keys after deletion: %v", err)
	}

	if len(foundKeys) != len(keys)-1 {
		t.Errorf("expected %d keys after deletion, got %d: %v", len(keys)-1, len(foundKeys), foundKeys)
	}

	for _, foundKey := range foundKeys {
		if foundKey == keys[1] {
			t.Errorf("deleted key %s should not be found in results %v", keys[1], foundKeys)
		}
	}
}

func TestKafkaStorage_PrefixFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kafka integration test in short mode")
		return
	}

	storage, err := NewStorage(kafkaStorageEndpoints)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.(*Storage).Close()

	ctx := context.Background()

	testData := map[string]string{
		"prefix1_key1": "value1",
		"prefix1_key2": "value2",
		"prefix2_key1": "value3",
		"prefix2_key2": "value4",
		"other_key":    "value5",
	}

	ttl := 30 * time.Second

	for key, value := range testData {
		err = storage.SetEx(ctx, key, value, ttl)
		if err != nil {
			t.Fatalf("failed to set key %s: %v", key, err)
		}
	}

	time.Sleep(2 * time.Second)

	prefix1Keys, err := storage.Keys(ctx, "prefix1_")
	if err != nil {
		t.Fatalf("failed to get prefix1 keys: %v", err)
	}

	if len(prefix1Keys) != 2 {
		t.Errorf("expected 2 prefix1 keys, got %d: %v", len(prefix1Keys), prefix1Keys)
	}

	prefix2Keys, err := storage.Keys(ctx, "prefix2_")
	if err != nil {
		t.Fatalf("failed to get prefix2 keys: %v", err)
	}

	if len(prefix2Keys) != 2 {
		t.Errorf("expected 2 prefix2 keys, got %d: %v", len(prefix2Keys), prefix2Keys)
	}

	otherKeys, err := storage.Keys(ctx, "other_")
	if err != nil {
		t.Fatalf("failed to get other keys: %v", err)
	}

	if len(otherKeys) != 1 {
		t.Errorf("expected 1 other key, got %d: %v", len(otherKeys), otherKeys)
	}
}

func TestKafkaStorage_ValueTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping kafka integration test in short mode")
		return
	}

	storage, err := NewStorage(kafkaStorageEndpoints)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer storage.(*Storage).Close()

	ctx := context.Background()
	ttl := 30 * time.Second

	testCases := []struct {
		key   string
		value interface{}
	}{
		{"string_key", "string_value"},
		{"bytes_key", []byte("bytes_value")},
		{"int_key", 12345},
		{"float_key", 123.45},
	}

	for _, tc := range testCases {
		err = storage.SetEx(ctx, tc.key, tc.value, ttl)
		if err != nil {
			t.Fatalf("failed to set key %s with value type %T: %v", tc.key, tc.value, err)
		}
	}

	time.Sleep(2 * time.Second)

	for _, tc := range testCases {
		keys, err := storage.Keys(ctx, tc.key)
		if err != nil {
			t.Fatalf("failed to get keys for %s: %v", tc.key, err)
		}

		found := false
		for _, k := range keys {
			if k == tc.key {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("key %s with value type %T not found", tc.key, tc.value)
		}
	}
}
