package genjson

import (
	"fmt"
	"os"
	"strings"
)

// 定义 DataX 配置结构
type DataXConfig struct {
	Job struct {
		Setting struct {
			Speed struct {
				Byte int `json:"byte"`
			} `json:"speed"`
			ErrorLimit struct {
				Record     int     `json:"record"`
				Percentage float64 `json:"percentage"`
			} `json:"errorLimit"`
		} `json:"setting"`
		Content []struct {
			Reader struct {
				Name      string `json:"name"`
				Parameter struct {
					Username   string `json:"username"`
					Password   string `json:"password"`
					SplitPk    string `json:"splitPk"`
					Connection []struct {
						QuerySql []string `json:"querySql"`
						JdbcUrl  []string `json:"jdbcUrl"`
					} `json:"connection"`
				} `json:"parameter"`
			} `json:"reader"`
			Writer struct {
				Name      string `json:"name"`
				Parameter struct {
					Username   string   `json:"username"`
					Password   string   `json:"password"`
					Column     []string `json:"column"`
					PreSql     []string `json:"preSql"`
					Connection []struct {
						JdbcUrl string   `json:"jdbcUrl"`
						Table   []string `json:"table"`
					} `json:"connection"`
				} `json:"parameter"`
			} `json:"writer"`
		} `json:"content"`
	} `json:"job"`
}

// 替换模板中的占位符
func ReplacePlaceholders(template string, placeholders map[string]string) string {
	for key, value := range placeholders {
		placeholder := fmt.Sprintf("{{%s}}", key)
		template = strings.ReplaceAll(template, placeholder, value)
	}
	return template
}

// 读取模板文件并替换占位符
func GenerateDataXConfig(templateFilePath string, placeholders map[string]string) (string, error) {
	// 读取模板文件内容
	template, err := os.ReadFile(templateFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template file: %v", err)
	}

	// 替换占位符
	updatedTemplate := ReplacePlaceholders(string(template), placeholders)

	// 返回更新后的模板内容
	return updatedTemplate, nil
}

// 保存更新后的配置文件
func SaveConfigToFile(configJSON, filePath string) error {
	return os.WriteFile(filePath, []byte(configJSON), 0644)
}
