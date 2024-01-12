COMMAND_MAIN:=cmd/main
EXE_FILE_NAME:=main.exe
DEFAULT_NETNS:=router1

ifdef NETNS
  OVERRIDE_NETNS := $(NETNS)
else
  OVERRIDE_NETNS := $(DEFAULT_NETNS)
endif

build:
	@ip netns exec ${OVERRIDE_NETNS} go build -o ./$(COMMAND_MAIN)/${EXE_FILE_NAME} ./$(COMMAND_MAIN)
run:
	@ip netns exec ${OVERRIDE_NETNS} ./$(COMMAND_MAIN)/${EXE_FILE_NAME}
fmt:
	@go fmt ./cmd/*
