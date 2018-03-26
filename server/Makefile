run: 
	go run main.go
build: main.go
	go build ./...
test: 
	go test ./...
rpc: rpc/service.proto
	protoc -I$(GOPATH)/src/github.com/google/protobuf/src/google/protobuf -I$(GOPATH)/src/github.com/mikerjacobi/notify-app/server/rpc --go_out=./rpc --twirp_out=./rpc $(GOPATH)/src/github.com/mikerjacobi/notify-app/server/rpc/*.proto
db: sql
	mysql -hnotify.cs9ds6yfnikc.us-east-1.rds.amazonaws.com -udbuser -p$(shell cat /etc/secrets/notify-db.json | grep password | cut -d'"' -f4) -Dnotify < sql/make_dbs.sql
mysql:
	mysql -hnotify.cs9ds6yfnikc.us-east-1.rds.amazonaws.com -udbuser -p$(shell cat /etc/secrets/notify-db.json | grep password | cut -d'"' -f4) -Dnotify

