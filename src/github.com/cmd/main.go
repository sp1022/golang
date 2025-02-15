package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"transdbx/src/github.com/db"
	"transdbx/src/github.com/genjson"

	dataxr "github.com/koolay/datax-runner"
)

type StdoutLog struct {
}

type StderrLog struct {
}

func (lg *StdoutLog) Write(text string) {
	log.Println("[Stdout]", text)
}

func (lg *StderrLog) Write(text string) {
	log.Println("[Stderr]", text)
}

func main() {
	//1设置路径
	var cfgFilePath, dataxHome, sourceTable, TargetTable, sourceSchema, targetPart, targetPsql string
	//flag.StringVar(&cfgFilePath, "config", "./datax_pg_job.json", "job config file")
	flag.StringVar(&sourceTable, "T", "", "source table name")
	flag.StringVar(&TargetTable, "t", "", "target table name")
	flag.StringVar(&targetPart, "p", "", "part name of target table,only oracle!!!!")
	flag.StringVar(&sourceSchema, "schema", "", "table in  which schema")
	flag.StringVar(&dataxHome, "dataxhome", "", "job config file")
	flag.StringVar(&targetPsql, "presql", "", "job config file")
	flag.Parse()
	if sourceTable == "" {
		log.Fatalf("please set source table name using: -T")
		os.Exit(1)
	}
	if TargetTable == "" {
		log.Fatalf("please set target table name using: -t")
		os.Exit(1)
	}
	if sourceSchema == "" {
		log.Fatalf("please set target table name using: -schema")
		os.Exit(1)
	}
	if dataxHome == "" {
		log.Fatalf("please set datax home using: -dataxhome")
		os.Exit(1)
	}
	//2读取配置文件:模板以及db配置
	//2.1获取可执行文件所在的目录路径
	executable, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to exec: %v", err)
	}
	dir := filepath.Dir(executable)
	//dir = "C:\\Users\\ThinkPad\\Desktop\\golang"
	dbCfgPath := filepath.Join(dir, "/config/dbcfg.yaml")                  //数据库配置
	templateFilePath := filepath.Join(dir, "/config/datax_pg_config.json") //模板配置
	dbJson, err := db.ReadConfig(dbCfgPath)
	if err != nil {
		log.Fatalf("Failed to get json of db: %v", err)
	}

	//对于有多个source进行循环
	var syncGroup sync.WaitGroup
	for _, soureDBConf := range dbJson.Databases.SourceDBs {
		syncGroup.Add(1)
		go func() {
			defer syncGroup.Done()
			dbConn, err := db.GetDBConnection(soureDBConf)
			if err != nil {
				log.Fatalf("Failed to connect db: %v", err)
			}
			columns, err := db.GetTableColumns(dbConn, sourceSchema, sourceTable, soureDBConf.Dbtype)
			if err != nil {
				log.Fatalf("Failed to get column for db: %v", err)
			}
			// 格式化列名为 JSON 数组字符串
			// columnJSON, err := json.Marshal(columns)
			// if err != nil {
			// 	log.Fatalf("Error marshalling columns: %v", err)
			// }
			sourceColumnStr := fmt.Sprintf("%s", strings.Join(columns, `","`)) //注意格式
			//2自动生成模板文件
			//解析dbconfig文件

			//2.1根据表生成json文件，其中源端库和目标库都在配置文件中进行配置
			placeholders := map[string]string{
				"source_username": soureDBConf.User,
				"source_password": soureDBConf.Password,
				"source_table":    sourceTable,
				"source_columns":  strings.Join(columns, ","), // 动态插入列名,
				"source_db":       soureDBConf.Dbname,
				"source_host":     soureDBConf.Host,
				"source_port":     strconv.Itoa(soureDBConf.Port),
				"source_dbtype":   soureDBConf.Dbtype,
				"target_username": dbJson.Databases.TargetDB.User,
				"target_password": dbJson.Databases.TargetDB.Password,
				"target_table":    TargetTable,
				"target_columns":  sourceColumnStr, // 动态插入列名,
				"target_db":       dbJson.Databases.TargetDB.Dbname,
				"target_host":     dbJson.Databases.TargetDB.Host,
				"target_port":     strconv.Itoa(dbJson.Databases.TargetDB.Port),
				"target_dbtype":   dbJson.Databases.TargetDB.Dbtype,
				"target_partname": targetPart,
				"target_Psql":     targetPsql,
			}
			//设置oracle分区如何查询
			if dbJson.Databases.TargetDB.Dbtype == "oracle" && targetPart != "" {
				placeholders["target_partname"] = " partition (" + targetPart + ")" // 设置为 oracle 特定值
			}

			//设置如何设置预执行sql
			if targetPsql == "" {
				if dbJson.Databases.TargetDB.Dbtype == "oracle" && targetPart != "" {
					targetPsql = "alter table " + TargetTable + " truncate partition " + targetPart
					placeholders["target_Psql"] = targetPsql
				} else {
					targetPsql = "delete from " + TargetTable
				}
			}
			// 读取模板文件并替换占位符
			updatedConfig, err := genjson.GenerateDataXConfig(templateFilePath, placeholders)
			if err != nil {
				log.Fatalf("Error generating DataX config: %v", err)
			}

			//2.2保存配置到文件
			cfgFilePath = filepath.Join(dir, "/config/datax_pg_config_"+sourceTable+".json")
			err = genjson.SaveConfigToFile(updatedConfig, cfgFilePath)
			if err != nil {
				log.Fatalf("Error saving config to file: %v", err)
			}

			//3启动
			datax := dataxr.NewDataX(dataxr.Config{
				Debug:      true,
				Xms:        dbJson.Xms,
				Xmx:        dbJson.Xmx,
				Loglevel:   "debug",
				DataxHome:  dataxHome,
				Mode:       "",
				Jobid:      "1",
				ConfigFile: cfgFilePath,
			}, &StdoutLog{}, &StderrLog{})

			ctx := context.Background()

			pid, err := datax.Exec(ctx, "java")

			if err != nil {
				log.Fatal(err)
			}

			log.Println("pid", pid)

			err = datax.Wait(ctx, 3600*time.Second)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}
	syncGroup.Wait()
}
