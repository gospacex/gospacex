package mq

import (
	"context"
	"testing"
	"time"
)

func TestKafkaProducer_Partition(t *testing.T) {
	cfg := KafkaProducerConfig{
		Brokers:        []string{"localhost:9092"},
		TopicPrefix:    "test-logs",
		PartitionCount: 64,
		BatchSize:      100,
		FlushInterval:  time.Second,
	}

	producer, err := NewKafkaProducer(cfg)
	if err != nil {
		t.Skip("kafka not available, skipping test")
	}
	defer producer.Close()

	tests := []struct {
		name    string
		key     string
		want    int32
		wantErr bool
	}{
		{
			name:    "same key returns same partition",
			key:     "trace-123",
			want:    producer.partition("trace-123"),
			wantErr: false,
		},
		{
			name:    "different key may return different partition",
			key:     "trace-456",
			want:    -1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := producer.partition(tt.key)
			if tt.want != -1 && got != tt.want {
				t.Errorf("partition() = %v, want %v", got, tt.want)
			}
			if got < 0 || got >= producer.partitionCount {
				t.Errorf("partition() = %v, want between 0 and %v", got, producer.partitionCount-1)
			}
		})
	}
}

func TestKafkaProducer_ConsistentHashing(t *testing.T) {
	cfg := KafkaProducerConfig{
		Brokers:        []string{"localhost:9092"},
		TopicPrefix:    "test-logs",
		PartitionCount: 64,
		BatchSize:      100,
		FlushInterval:  time.Second,
	}

	producer, err := NewKafkaProducer(cfg)
	if err != nil {
		t.Skip("kafka not available, skipping test")
	}
	defer producer.Close()

	key := "trace-consistent-123"
	partitions := make(map[int32]int)

	for i := 0; i < 100; i++ {
		p := producer.partition(key)
		partitions[p]++
	}

	if len(partitions) != 1 {
		t.Errorf("consistent hashing failed: expected 1 partition, got %d", len(partitions))
	}
}

func TestKafkaProducer_Push(t *testing.T) {
	cfg := KafkaProducerConfig{
		Brokers:        []string{"localhost:9092"},
		TopicPrefix:    "test-logs",
		PartitionCount: 64,
		BatchSize:      100,
		FlushInterval:  time.Second,
	}

	producer, err := NewKafkaProducer(cfg)
	if err != nil {
		t.Skip("kafka not available, skipping test")
	}
	defer producer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = producer.Push(ctx, "business", "key-1", []byte(`{"test":"data"}`))
	if err != nil {
		t.Errorf("Push() error = %v", err)
	}
}
