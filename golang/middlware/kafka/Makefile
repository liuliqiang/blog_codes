.PHONY: consumer producer

consumer:
	go run consumer/main.go \
		-brokers 192.168.77.81:9092,192.168.77.82:9092,192.168.77.83:9092 \
		-group test \
		-topics liuliqiang-test

producer:
	go run producer/main.go \
		-brokers 192.168.77.81:9092,192.168.77.82:9092,192.168.77.83:9092 \
		-topic liuliqiang-test