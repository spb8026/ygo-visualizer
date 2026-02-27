package ygopenpb

//go:generate protoc --proto_path=../../ygopen/include/ygopen/proto --go_out=. --go_opt=paths=source_relative --go_opt=Mdeck.proto=github.com/spb8026/ygo-visualizer/ygopenpb --go_opt=Mbanlist.proto=github.com/spb8026/ygo-visualizer/ygopenpb --go_opt=Mduel_data.proto=github.com/spb8026/ygo-visualizer/ygopenpb --go_opt=Mduel_msg.proto=github.com/spb8026/ygo-visualizer/ygopenpb --go_opt=Mduel_answer.proto=github.com/spb8026/ygo-visualizer/ygopenpb duel_data.proto duel_msg.proto duel_answer.proto
