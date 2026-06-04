package kafka

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"openreplay/backend/pkg/logger"
	"openreplay/backend/pkg/messages"
	"openreplay/backend/pkg/queue/types"
)

// KafkaConsumer combines a consumer-group (group.id, broker-managed offsets)
// with manual partition assignment via Assign(), so each pod is sticky to a
// hostname-derived partition while offsets are still committed to the group.
//
// segmentio/kafka-go rejected this combination at config validation:
//
//	if config.GroupID != "" {
//	    if config.Partition != 0 {
//	        return errors.New("either Partition or GroupID may be specified, but not both")
//	    }
//	}
//
// confluent-kafka-go (librdkafka) supports it by skipping Subscribe() entirely
// and calling Assign() on a consumer that has group.id configured.
type KafkaConsumer struct {
	consumer         *kafka.Consumer
	logger           logger.Logger
	iterator         messages.MessageIterator
	topic            string
	pendingMessages  map[int32][]*kafka.Message
	lastTs           map[int32]int64
	rebalanceHandler types.RebalanceHandler
	partitions       []int
}

// GetPartitionFromHostname extracts the integer suffix from $HOSTNAME (e.g.
// "lenz-worker-3" → 3). It powers the sticky partition mapping for stateful
// pods.
func GetPartitionFromHostname() (int, error) {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		return 0, fmt.Errorf("kafka: HOSTNAME is empty; set kafkaConsumerPartition in config for local dev")
	}
	parts := strings.Split(hostname, "-")
	p, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, fmt.Errorf("kafka: HOSTNAME %q has no numeric suffix (expected e.g. ender-0); set kafkaConsumerPartition", hostname)
	}
	return p, nil
}

func NewKafkaConsumer(
	log logger.Logger,
	brokers []string,
	group string,
	topic string,
	partitions []int,
	iterator messages.MessageIterator,
	clientCfg ClientConfig,
	rebalanceHandler types.RebalanceHandler,
) (*KafkaConsumer, error) {
	if group == "" {
		return nil, errors.New("kafka: groupId is empty")
	}
	if len(brokers) == 0 {
		return nil, errors.New("kafka: brokers is empty")
	}
	if len(partitions) == 0 {
		return nil, errors.New("kafka: partitions is empty")
	}
	if topic == "" {
		return nil, errors.New("kafka: topic is empty")
	}

	cm, err := BuildConfigMap(JoinBootstrapServers(brokers), clientCfg, map[string]any{
		"group.id":           group,
		"enable.auto.commit": false,
		"auto.offset.reset":  "earliest",
		"session.timeout.ms": 30000,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka consumer config: %w", err)
	}
	c, err := kafka.NewConsumer(cm)
	if err != nil {
		return nil, fmt.Errorf("kafka.NewConsumer: %w", err)
	}

	tps := make([]kafka.TopicPartition, 0, len(partitions))
	topicCopy := topic
	for _, p := range partitions {
		tps = append(tps, kafka.TopicPartition{
			Topic:     &topicCopy,
			Partition: int32(p),
			Offset:    kafka.OffsetStored,
		})
	}
	if err := c.Assign(tps); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("kafka Assign: %w", err)
	}

	if rebalanceHandler != nil {
		ups := make([]uint64, 0, len(partitions))
		for _, p := range partitions {
			ups = append(ups, uint64(p))
		}
		rebalanceHandler(types.RebalanceTypeAssign, ups)
	}

	return &KafkaConsumer{
		consumer:         c,
		logger:           log,
		iterator:         iterator,
		topic:            topic,
		pendingMessages:  make(map[int32][]*kafka.Message),
		lastTs:           make(map[int32]int64),
		rebalanceHandler: rebalanceHandler,
		partitions:       partitions,
	}, nil
}

func (kc *KafkaConsumer) ConsumeNext() error {
	msg, err := kc.consumer.ReadMessage(1 * time.Second)
	if err != nil {
		var kerr kafka.Error
		if errors.As(err, &kerr) && kerr.IsTimeout() {
			return nil
		}
		return err
	}

	var sessionID uint64
	if len(msg.Key) >= 8 {
		sessionID = binary.BigEndian.Uint64(msg.Key)
	}

	partition := msg.TopicPartition.Partition
	kc.pendingMessages[partition] = append(kc.pendingMessages[partition], msg)

	ts := msg.Timestamp.UnixMilli()
	if ts > kc.lastTs[partition] {
		kc.lastTs[partition] = ts
	}

	topicName := ""
	if msg.TopicPartition.Topic != nil {
		topicName = *msg.TopicPartition.Topic
	}

	kc.logger.Info(context.Background(),
		"pod=%s consuming topic=%s partition=%d offset=%d",
		os.Getenv("HOSTNAME"),
		topicName,
		partition,
		int64(msg.TopicPartition.Offset),
	)

	kc.iterator.Iterate(
		msg.Value,
		messages.NewBatchInfo(
			sessionID,
			topicName,
			uint64(msg.TopicPartition.Offset),
			uint64(partition),
			ts,
		),
	)
	return nil
}

// commitUpTo commits the highest offset across msgs for that partition. Kafka
// offsets are "next-to-consume", hence offset+1.
func (kc *KafkaConsumer) commitUpTo(msgs []*kafka.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	last := msgs[0]
	for _, m := range msgs[1:] {
		if m.TopicPartition.Offset > last.TopicPartition.Offset {
			last = m
		}
	}
	tp := kafka.TopicPartition{
		Topic:     last.TopicPartition.Topic,
		Partition: last.TopicPartition.Partition,
		Offset:    last.TopicPartition.Offset + 1,
	}
	if _, err := kc.consumer.CommitOffsets([]kafka.TopicPartition{tp}); err != nil {
		return err
	}
	return nil
}

func (kc *KafkaConsumer) CommitBack(gap int64) error {
	for partition, msgs := range kc.pendingMessages {
		if len(msgs) == 0 {
			continue
		}
		lastTs := kc.lastTs[partition]
		if lastTs == 0 {
			continue
		}
		maxTs := lastTs - gap

		var toCommit, remaining []*kafka.Message
		for _, m := range msgs {
			if m.Timestamp.UnixMilli() <= maxTs {
				toCommit = append(toCommit, m)
			} else {
				remaining = append(remaining, m)
			}
		}
		if err := kc.commitUpTo(toCommit); err != nil {
			return err
		}
		kc.pendingMessages[partition] = remaining
	}
	return nil
}

func (kc *KafkaConsumer) Commit() error {
	for partition, msgs := range kc.pendingMessages {
		if len(msgs) == 0 {
			continue
		}
		if err := kc.commitUpTo(msgs); err != nil {
			return err
		}
		kc.pendingMessages[partition] = nil
	}
	return nil
}

func (kc *KafkaConsumer) Close() {
	if kc.consumer == nil {
		return
	}
	if kc.rebalanceHandler != nil {
		ups := make([]uint64, 0, len(kc.partitions))
		for _, p := range kc.partitions {
			ups = append(ups, uint64(p))
		}
		kc.rebalanceHandler(types.RebalanceTypeRevoke, ups)
	}
	_ = kc.consumer.Close()
}
