.PHONY: run_db install_mysql_client connect_db prepare_db run_bank1 run_bank2 tran_00

DB_VERSION=5.7

run_db:
	@echo "Starting database..."
	@docker run -d --name liuliqiang_mysql \
		-p 3306:3306 \
		-e MYSQL_ROOT_PASSWORD=password \
		-e MYSQL_DATABASE=lq_test \
		mysql:$(DB_VERSION)

install_mysql_client:
	@echo "Installing mysql client..."
	# @sudo apt-get install mysql-client
	@sudo yum install mysql

connect_db:
	@echo "Connecting to database..."
	@mysql -h127.0.0.1 -uroot -ppassword lq_test

prepare_db:
	@echo "Preparing database..."
	@mysql -h127.0.0.1 -uroot -ppassword lq_test < ./scripts/test_00.sql


run_bank1:
	@echo "Starting bank1..."
	@go run ./bank01/main.go -name citi_bank -port 9010

run_bank2:
	@echo "Starting bank2..."
	@go run ./bank02/main.go -name hsbc_bank -port 9020

tran_00:
	@echo "Transfering money from citi to hsbc..."
	@go run ./transaction/cross_bank/main.go 