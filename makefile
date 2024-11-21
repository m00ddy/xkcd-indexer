# Define the Go test command with a placeholder for the test function
TEST_CMD = go test -v -tags "fts5" -run $(TEST) -count=1 ./$(DIR)

# Default target
.PHONY: test
test:
	@echo "Running tests with function: $(TEST)"
	$(TEST_CMD)
