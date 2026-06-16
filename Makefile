help:
	@echo "Usage: make <command>\n"
	@echo "Commands:"
	@printf "%3sstart-testdb: Starts the test SQL Docker container\n"
	@printf "%3sstop-testdb: Stops the test SQL Docker container\n"
	@printf "%3srestart-testdb: Restarts the test SQL docker container\n"
	@printf "%3snpm-rundev: Starts the frontend developmental server\n"
	@printf "%3snpm-install: Installs npm dependencies\n"
	@exit 0

start-testdb:
	@bash ./tools/start_testdb.sh

stop-testdb:
	@bash ./tools/stop_testdb.sh

restart-testdb:
	@bash ./tools/stop_testdb.sh
	@bash ./tools/start_testdb.sh

npm-rundev:
	@bash ./tools/npm_run.sh "./frontend"

npm-install:
	@bash ./tools/npm_install.sh