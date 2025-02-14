SRC_DIR=protobufs
DST_DIR=pkg/

protoc -I=$SRC_DIR --go_out=$DST_DIR message.proto
