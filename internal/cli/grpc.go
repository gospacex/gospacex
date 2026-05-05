package cli

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
)

// gRPC 生成器标志
var (
	grpcDBDSN       string
	grpcTables      string
	grpcIDLPath     string
	grpcSrvPath     string
	grpcBFFPath     string
	grpcProtoImport string
	grpcDryRun      bool
	grpcSrvPort     int
	grpcBFFPort     int
)

// GRPCTableInfo 表信息
type GRPCTableInfo struct {
	Name       string
	GoName     string
	Comment    string
	Columns    []GRPCColumnInfo
	PrimaryKey string
}

// GRPCColumnInfo 列信息
type GRPCColumnInfo struct {
	Name      string
	GoName    string
	GoType    string
	ProtoType string
	IsPrimary bool
	Comment   string
}

var genGRPCCmd = &cobra.Command{
	Use:   "gen-grpc",
	Short: "从数据库表结构生成 gRPC 代码",
	Long: `从数据库表结构自动生成完整的 gRPC 服务端和客户端代码。

示例：
  gpx gen-grpc --db-dsn="root:password@tcp(localhost:3306)/mydb"
  gpx gen-grpc --db-dsn="..." --tables="users,orders"
  gpx gen-grpc --db-dsn="..." --dry-run`,
	RunE: runGenGRPC,
}

func runGenGRPC(cmd *cobra.Command, args []string) error {
	if grpcDBDSN == "" {
		return fmt.Errorf("--db-dsn is required")
	}

	// 解析 DSN
	host, port, user, password, dbName := parseDSN(grpcDBDSN)

	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}
	defer db.Close()

	// 获取表列表
	tables := strings.Split(grpcTables, ",")
	if grpcTables == "" {
		tables, err = getAllTables(db)
		if err != nil {
			return fmt.Errorf("failed to get tables: %w", err)
		}
	}

	fmt.Printf("Generating gRPC code for %d tables...\n", len(tables))

	for _, tableName := range tables {
		tableName = strings.TrimSpace(tableName)
		if tableName == "" {
			continue
		}

		// 获取表信息
		tableInfo, err := getGRPCTableInfo(db, tableName)
		if err != nil {
			fmt.Printf("Warning: failed to get info for table %s: %v\n", tableName, err)
			continue
		}

		// 生成 Proto 文件
		if grpcIDLPath != "" {
			if err := generateProtoFile(tableInfo); err != nil {
				fmt.Printf("Warning: failed to generate proto for %s: %v\n", tableName, err)
			}
		}

		// 生成微服务代码
		if grpcSrvPath != "" {
			if err := generateSrvGRPC(tableInfo); err != nil {
				fmt.Printf("Warning: failed to generate service for %s: %v\n", tableName, err)
			}
		}

		// 生成 BFF 代码
		if grpcBFFPath != "" {
			if err := generateBFFGRPC(tableInfo); err != nil {
				fmt.Printf("Warning: failed to generate BFF for %s: %v\n", tableName, err)
			}
		}

		fmt.Printf("✓ Generated gRPC code for table: %s\n", tableName)
	}

	fmt.Println("\ngRPC code generation completed!")
	if grpcDryRun {
		fmt.Println("(Dry run mode - no files written)")
	}

	return nil
}

func parseDSN(dsn string) (host, port, user, password, dbName string) {
	// 格式: user:password@tcp(host:port)/dbname
	dsn = strings.TrimPrefix(dsn, "mysql://")
	dsn = strings.TrimPrefix(dsn, "root:")

	parts := strings.Split(dsn, "@")
	if len(parts) >= 2 {
		authParts := strings.Split(parts[0], ":")
		if len(authParts) >= 2 {
			user = authParts[0]
			password = authParts[1]
		}
		rest := parts[1]
		dbParts := strings.Split(rest, "/")
		if len(dbParts) >= 2 {
			hostPort := dbParts[0]
			dbName = dbParts[1]
			hpParts := strings.Split(hostPort, ":")
			host = hpParts[0]
			if len(hpParts) >= 2 {
				port = hpParts[1]
			}
		}
	}
	return
}

func getAllTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, nil
}

func getGRPCTableInfo(db *sql.DB, tableName string) (*GRPCTableInfo, error) {
	// 获取列信息
	query := `SELECT COLUMN_NAME, DATA_TYPE, COLUMN_KEY, COLUMN_COMMENT, IS_NULLABLE, COLUMN_DEFAULT
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	info := &GRPCTableInfo{
		Name:    tableName,
		GoName:  toGoName(tableName),
		Columns: []GRPCColumnInfo{},
	}

	for rows.Next() {
		var colName, dataType, key, comment, nullable, defaultVal string
		if err := rows.Scan(&colName, &dataType, &key, &comment, &nullable, &defaultVal); err != nil {
			continue
		}
		col := GRPCColumnInfo{
			Name:      colName,
			GoName:    toGoName(colName),
			GoType:    getGoType(dataType),
			ProtoType: getProtoType(dataType),
			IsPrimary: key == "PRI",
			Comment:   comment,
		}
		info.Columns = append(info.Columns, col)
		if key == "PRI" {
			info.PrimaryKey = colName
		}
	}

	return info, nil
}

func toGoName(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(string(parts[i][0])) + parts[i][1:]
		}
	}
	result := strings.Join(parts, "")
	return strings.ToUpper(result[:1]) + result[1:]
}

func getGoType(dbType string) string {
	dbType = strings.ToLower(dbType)
	switch {
	case strings.HasPrefix(dbType, "bigint"), strings.HasPrefix(dbType, "int"):
		return "int64"
	case strings.HasPrefix(dbType, "float"), strings.HasPrefix(dbType, "double"), strings.HasPrefix(dbType, "decimal"):
		return "float64"
	case strings.HasPrefix(dbType, "varchar"), strings.HasPrefix(dbType, "char"), strings.HasPrefix(dbType, "text"):
		return "string"
	case strings.HasPrefix(dbType, "datetime"), strings.HasPrefix(dbType, "timestamp"), strings.HasPrefix(dbType, "date"):
		return "time.Time"
	case strings.HasPrefix(dbType, "blob"):
		return "[]byte"
	case strings.HasPrefix(dbType, "bit"), strings.HasPrefix(dbType, "bool"):
		return "bool"
	default:
		return "string"
	}
}

func getProtoType(dbType string) string {
	dbType = strings.ToLower(dbType)
	switch {
	case strings.HasPrefix(dbType, "bigint"), strings.HasPrefix(dbType, "int"):
		return "int64"
	case strings.HasPrefix(dbType, "float"):
		return "float"
	case strings.HasPrefix(dbType, "double"), strings.HasPrefix(dbType, "decimal"):
		return "double"
	case strings.HasPrefix(dbType, "varchar"), strings.HasPrefix(dbType, "char"), strings.HasPrefix(dbType, "text"):
		return "string"
	case strings.HasPrefix(dbType, "datetime"), strings.HasPrefix(dbType, "timestamp"), strings.HasPrefix(dbType, "date"):
		return "int64"
	case strings.HasPrefix(dbType, "blob"):
		return "bytes"
	case strings.HasPrefix(dbType, "bit"), strings.HasPrefix(dbType, "bool"):
		return "bool"
	default:
		return "string"
	}
}

func generateProtoFile(info *GRPCTableInfo) error {
	if grpcDryRun {
		fmt.Printf("  [DRY RUN] Would generate proto: %s.proto\n", info.Name)
		return nil
	}

	g := info.GoName
	protoContent := `// Code generated by gospacex. DO NOT EDIT.

syntax = "proto3";

package ` + g + `;

option go_package = "github.com/example/` + grpcIDLPath + `/` + info.Name + `";

service ` + g + `Service {
    rpc Create(` + g + `Request) returns (` + g + `Response);
    rpc Update(` + g + `Request) returns (` + g + `Response);
    rpc Delete(` + g + `Request) returns (` + g + `Response);
    rpc GetByID(` + g + `Request) returns (` + g + `Response);
    rpc List(` + g + `Request) returns (` + g + `ListResponse);
    rpc Page(` + g + `Request) returns (` + g + `PageResponse);
}

message ` + g + ` {
`

	protoContent += fmt.Sprintf("    // %s\n", info.Comment)

	for i, col := range info.Columns {
		num := i + 1
		protoContent += fmt.Sprintf("    %s %s = %d; // %s\n", col.ProtoType, col.GoName, num, col.Comment)
	}

	protoContent += "}\n\n"

	protoContent += `message ` + g + `Request {
    string request_id = 1;
`

	for _, col := range info.Columns {
		if !col.IsPrimary {
			protoContent += fmt.Sprintf("    %s %s = %d;\n", col.ProtoType, col.GoName, 2)
		}
	}
	protoContent += "}\n\n"

	protoContent += `message ` + g + `Response {
    int32 code = 1;
    string message = 2;
    ` + g + ` data = 3;
}

message ` + g + `ListResponse {
    int32 code = 1;
    string message = 2;
    repeated ` + g + ` data_list = 3;
    int32 total = 4;
}

message ` + g + `PageResponse {
    int32 code = 1;
    string message = 2;
    repeated ` + g + ` data_list = 3;
    int64 total = 4;
    int32 page = 5;
    int32 page_size = 6;
}
`

	protoPath := filepath.Join(grpcIDLPath, info.Name+".proto")
	if err := os.MkdirAll(filepath.Dir(protoPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(protoPath, []byte(protoContent), 0644)
}

func generateSrvGRPC(info *GRPCTableInfo) error {
	if grpcDryRun {
		fmt.Printf("  [DRY RUN] Would generate service: %s_grpc.go\n", info.Name)
		return nil
	}

	importPath := grpcProtoImport
	if importPath == "" {
		importPath = fmt.Sprintf("github.com/example/%s/%s", filepath.Base(grpcSrvPath), info.Name)
	}

	g := info.GoName
	content := `// +build !gen

package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "` + importPath + `"
)

type ` + g + `Server struct {
	pb.Unimplemented` + g + `ServiceServer
}

func New` + g + `Server() *` + g + `Server {
	return &` + g + `Server{}
}

func (s *` + g + `Server) Create(ctx context.Context, req *pb.` + g + `Request) (*pb.` + g + `Response, error) {
	log.Printf("Create called: %v", req)
	return &pb.` + g + `Response{Code: 0, Message: "success", Data: &pb.` + g + `{}}, nil
}

func (s *` + g + `Server) Update(ctx context.Context, req *pb.` + g + `Request) (*pb.` + g + `Response, error) {
	log.Printf("Update called: %v", req)
	return &pb.` + g + `Response{Code: 0, Message: "success", Data: &pb.` + g + `{}}, nil
}

func (s *` + g + `Server) Delete(ctx context.Context, req *pb.` + g + `Request) (*pb.` + g + `Response, error) {
	log.Printf("Delete called: %v", req)
	return &pb.` + g + `Response{Code: 0, Message: "success"}, nil
}

func (s *` + g + `Server) GetByID(ctx context.Context, req *pb.` + g + `Request) (*pb.` + g + `Response, error) {
	log.Printf("GetByID called: %v", req)
	return &pb.` + g + `Response{Code: 0, Message: "success", Data: &pb.` + g + `{}}, nil
}

func (s *` + g + `Server) List(ctx context.Context, req *pb.` + g + `Request) (*pb.` + g + `ListResponse, error) {
	log.Printf("List called: %v", req)
	return &pb.` + g + `ListResponse{Code: 0, Message: "success"}, nil
}

func (s *` + g + `Server) Page(ctx context.Context, req *pb.` + g + `Request) (*pb.` + g + `PageResponse, error) {
	log.Printf("Page called: %v", req)
	return &pb.` + g + `PageResponse{Code: 0, Message: "success"}, nil
}

func main() {
	port := ` + fmt.Sprintf("%d", grpcSrvPort) + `
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.Register` + g + `ServiceServer(s, New` + g + `Server())
	reflection.Register(s)
	log.Printf("gRPC server listening on port %d", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
`

	srvDir := filepath.Join(grpcSrvPath, "srv_"+info.Name)
	if err := os.MkdirAll(srvDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(srvDir, info.Name+"_grpc.go"), []byte(content), 0644)
}

func generateBFFGRPC(info *GRPCTableInfo) error {
	if grpcDryRun {
		fmt.Printf("  [DRY RUN] Would generate BFF: handler_%s.go\n", info.Name)
		return nil
	}

	importPath := grpcProtoImport
	if importPath == "" {
		importPath = fmt.Sprintf("github.com/example/%s/%s", filepath.Base(grpcBFFPath), info.Name)
	}

	g := info.GoName

	// 生成 gRPC 客户端
	clientContent := `// +build !gen

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "` + importPath + `"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	client pb.` + g + `ServiceClient
}

func NewGRPCClient(addr string) (*GRPCClient, error) {
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(loggingInterceptor),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	return &GRPCClient{conn: conn, client: pb.New` + g + `ServiceClient(conn)}, nil
}

func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

func loggingInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	log.Printf("gRPC Call: %s, Request: %v", method, req)
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	log.Printf("gRPC Response: %s, Duration: %v", method, time.Since(start))
	return err
}

func (c *GRPCClient) Create(ctx context.Context, req *pb.` + g + `Request) (*pb.` + g + `Response, error) {
	return c.client.Create(ctx, req)
}

func (c *GRPCClient) GetByID(ctx context.Context, req *pb.` + g + `Request) (*pb.` + g + `Response, error) {
	return c.client.GetByID(ctx, req)
}

func (c *GRPCClient) List(ctx context.Context, req *pb.` + g + `Request) (*pb.` + g + `ListResponse, error) {
	return c.client.List(ctx, req)
}
`

	clientPath := filepath.Join(grpcBFFPath, "internal", "rpc", "grpc_client_"+info.Name+".go")
	if err := os.MkdirAll(filepath.Dir(clientPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(clientPath, []byte(clientContent), 0644); err != nil {
		return err
	}

	// 生成 HTTP Handler
	lowerName := strings.ToLower(info.Name[:1]) + info.Name[1:]
	handlerContent := fmt.Sprintf(`// +build !gen

package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

type %sHandler struct {
	client *GRPCClient
}

func New%sHandler(client *GRPCClient) *%sHandler {
	return &%sHandler{client: client}
}

func (h *%sHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/%s")
	{
		g.POST("", h.Create)
		g.GET("/:id", h.GetByID)
		g.GET("", h.List)
	}
}

func (h *%sHandler) Create(c *gin.Context) {
	var req pb.%sRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	resp, err := h.client.Create(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *%sHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	req := &pb.%sRequest{RequestId: id}
	resp, err := h.client.GetByID(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *%sHandler) List(c *gin.Context) {
	req := &pb.%sRequest{}
	resp, err := h.client.List(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
`,
		info.GoName, info.GoName, info.GoName, info.GoName,
		info.GoName, lowerName,
		info.GoName, info.GoName,
		info.GoName, info.GoName,
		info.GoName, info.GoName)

	handlerPath := filepath.Join(grpcBFFPath, "internal", "handler", "handler_"+info.Name+".go")
	if err := os.MkdirAll(filepath.Dir(handlerPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(handlerPath, []byte(handlerContent), 0644)
}

func GetGenGRPCCmd() *cobra.Command {
	return genGRPCCmd
}

func init() {
	genGRPCCmd.Flags().StringVar(&grpcDBDSN, "db-dsn", "", "数据库连接字符串 (user:password@tcp(host:port)/dbname)")
	genGRPCCmd.Flags().StringVar(&grpcTables, "tables", "", "指定表名，逗号分隔")
	genGRPCCmd.Flags().StringVar(&grpcIDLPath, "idl-path", "", "Proto 文件输出路径")
	genGRPCCmd.Flags().StringVar(&grpcSrvPath, "srv-path", "", "微服务代码输出路径")
	genGRPCCmd.Flags().StringVar(&grpcBFFPath, "bff-path", "", "BFF 层代码输出路径")
	genGRPCCmd.Flags().StringVar(&grpcProtoImport, "proto-import", "", "Proto 导入路径")
	genGRPCCmd.Flags().IntVar(&grpcSrvPort, "srv-port", 50051, "微服务 gRPC 端口")
	genGRPCCmd.Flags().IntVar(&grpcBFFPort, "bff-port", 8080, "BFF HTTP 端口")
	genGRPCCmd.Flags().BoolVar(&grpcDryRun, "dry-run", false, "预览模式，不写入文件")
}
