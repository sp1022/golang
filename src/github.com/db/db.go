package db

import (
	"database/sql"
	"fmt"
	"time"

	"io/ioutil"

	"gopkg.in/yaml.v2"

	_ "github.com/lib/pq"
)

// DBConfig 存储单个数据库的配置信息
type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Dbname   string `yaml:"dbname"`
	Dbtype   string `yaml:"dbtype"`
}

// Config 存储多个源数据库和目标数据库的配置
type Config struct {
	Databases struct {
		SourceDBs map[string]DBConfig `yaml:"source"`
		TargetDB  DBConfig            `yaml:"target"`
	} `yaml:"databases"`
}

// ReadConfig 从 YAML 文件中读取配置
func ReadConfig(configPath string) (*Config, error) {
	var config Config

	// 读取 YAML 配置文件
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}
	// 解析 YAML 数据
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}
	return &config, nil
}

// GetDBConnection 根据配置动态获取数据库连接
func GetDBConnection(dbConfig DBConfig) (*sql.DB, error) {
	var db *sql.DB
	var err error

	// 根据数据库类型选择连接方式
	switch dbConfig.Dbtype {
	case "postgresql":
		// 构造 PostgreSQL 连接字符串
		connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
			dbConfig.User, dbConfig.Password, dbConfig.Dbname, dbConfig.Host, dbConfig.Port)

		// 连接 PostgreSQL 数据库
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL: %v", err)
		}

	case "oracle":
		// 构造 Oracle 连接字符串
		connStr := fmt.Sprintf("%s/%s@%s:%d/%s", dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Dbname)

		// 连接 Oracle 数据库
		db, err = sql.Open("godror", connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Oracle: %v", err)
		}

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbConfig.Dbtype)
	}

	// 设置连接最大空闲时间，避免连接泄漏
	db.SetConnMaxLifetime(time.Minute * 5)
	// 设置连接池最大连接数
	db.SetMaxOpenConns(20)
	// 设置连接池最大空闲连接数
	db.SetMaxIdleConns(5)

	// 尝试连接数据库，确保数据库连接有效
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return db, nil
}

// GetTableColumns 获取 PostgreSQL 表的列名
func GetTableColumns(db *sql.DB, schemaName string, tableName string, dbType string) ([]string, error) {
	query := `
	 SELECT column_name
	 FROM information_schema.columns
	 WHERE table_schema = $1 AND table_name = $2
	 `
	switch dbType {
	case "oracle":
		query = `
	SELECT column_name
	FROM dba_tab_columns
	WHERE owner = $1 AND table_name = $2
	`
	case "postgresql":
	default:
	}

	rows, err := db.Query(query, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		columns = append(columns, columnName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}
