COMMAND_MAIN:=cmd/main
EXE_FILE_NAME:=main.exe
DEFAULT_NETNS:=router1

build:
	@ip netns exec ${DEFAULT_NETNS} go build -o ./$(COMMAND_MAIN)/${EXE_FILE_NAME} ./$(COMMAND_MAIN)
run:
	@ip netns exec ${DEFAULT_NETNS} ./$(COMMAND_MAIN)/${EXE_FILE_NAME}
fmt:
	@go fmt ./cmd/*
