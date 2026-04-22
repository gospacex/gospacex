# myshop

Microservice project.

## Options

- **Protocol**: grpc
- **HTTP**: gin
- **IDL**: proto

## Structure
```
myshop/
├── bffH5/
├── srvProduct/
├── common/
├── pkg/
└── scripts/
```

## Build
```bash
go mod init github.com/yourorg/myshop
./scripts/gen_proto.sh
./scripts/build.sh
```
