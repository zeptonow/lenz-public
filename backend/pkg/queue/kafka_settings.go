package queue

// KafkaSettingsFromProducerConsumer builds settings from resolved producer/consumer strings.
func KafkaSettingsFromProducerConsumer(
	producerBrokers, producerUser, producerPass string,
	consumerBrokers, consumerUser, consumerPass string,
	consumerPartition *int,
) KafkaSettings {
	return KafkaSettings{
		ProducerBrokers:      producerBrokers,
		ProducerSaslUsername: producerUser,
		ProducerSaslPassword: producerPass,
		ConsumerBrokers:      consumerBrokers,
		ConsumerSaslUsername: consumerUser,
		ConsumerSaslPassword: consumerPass,
		ConsumerPartition:    consumerPartition,
	}
}
