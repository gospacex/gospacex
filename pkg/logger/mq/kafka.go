package mq

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type KafkaProducer struct {
	brokers        []string
	topicPrefix    string
	partitionCount int32
	asyncProducer  sarama.AsyncProducer
	healthy        atomic.Bool
	closeCh        chan struct{}
	closeOnce      sync.Once
	logger         *zap.Logger
	errorLogger    *zap.Logger
}

func NewKafkaProducer(cfg KafkaProducerConfig) (*KafkaProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForLocal
	saramaConfig.Producer.Compression = sarama.CompressionSnappy
	saramaConfig.Producer.Flush.Frequency = cfg.FlushInterval
	saramaConfig.Producer.Flush.Messages = cfg.BatchSize
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	producer, err := sarama.NewAsyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	errorLogPath := filepath.Join("logs", "error_mq.log")
	if err := os.MkdirAll(filepath.Dir(errorLogPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create error log dir: %w", err)
	}
	errorFile, err := os.OpenFile(errorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open error log file: %w", err)
	}
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	var errorLogger *zap.Logger
	errorLogger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(errorFile),
		zapcore.WarnLevel,
	))

	p := &KafkaProducer{
		brokers:        cfg.Brokers,
		topicPrefix:    cfg.TopicPrefix,
		partitionCount: cfg.PartitionCount,
		asyncProducer:  producer,
		healthy:        atomic.Bool{},
		closeCh:        make(chan struct{}),
		logger:         zap.L().Named("kafka"),
		errorLogger:    errorLogger,
	}

	p.healthy.Store(true)
	go p.watchErrors()
	return p, nil
}

func (p *KafkaProducer) watchErrors() {
	for {
		select {
		case <-p.closeCh:
			return
		case err := <-p.asyncProducer.Errors():
			if err != nil {
				p.logger.Error("kafka producer error", zap.Error(err))
				p.retryWithBackoff(err)
			}
		}
	}
}

func (p *KafkaProducer) retryWithBackoff(producerErr *sarama.ProducerError) {
	backoffs := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	for _, backoff := range backoffs {
		time.Sleep(backoff)
		select {
		case <-p.closeCh:
			return
		default:
		}
	}
	p.logger.Error("kafka push failed after retries", zap.Error(producerErr.Err))

	topic := ""
	partition := int32(-1)
	var key string
	if producerErr.Msg != nil {
		topic = producerErr.Msg.Topic
		partition = producerErr.Msg.Partition
		if producerErr.Msg.Key != nil {
			if k, err := producerErr.Msg.Key.Encode(); err == nil {
				key = string(k)
			}
		}
	}
	p.errorLogger.Error("mq push failed after retries",
		zap.String("topic", topic),
		zap.Int32("partition", partition),
		zap.String("key", key),
		zap.Error(producerErr.Err),
	)
}

func (p *KafkaProducer) Push(ctx context.Context, scene, key string, data []byte) error {
	if !p.healthy.Load() {
		return fmt.Errorf("kafka producer is unhealthy")
	}

	topic := p.topicPrefix + "-" + scene

	_ = p.partition(key)

	msg := &sarama.ProducerMessage{
		Topic:    topic,
		Key:      sarama.StringEncoder(key),
		Value:    sarama.ByteEncoder(data),
		Metadata: scene,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.asyncProducer.Input() <- msg:
		return nil
	}
}

func (p *KafkaProducer) partition(key string) int32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int32(h.Sum32() % uint32(p.partitionCount))
}

func (p *KafkaProducer) Healthy() bool {
	return p.healthy.Load()
}

func (p *KafkaProducer) Close() error {
	p.closeOnce.Do(func() {
		close(p.closeCh)
	})
	return p.asyncProducer.Close()
}

type KafkaProducerConfig struct {
	Brokers        []string
	TopicPrefix    string
	PartitionCount int32
	BatchSize      int
	FlushInterval  time.Duration
}
