package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/Lumos-Labs-HQ/graft/internal/config"
	"github.com/Lumos-Labs-HQ/graft/internal/utils"
)

var (
	createTableRegex *regexp.Regexp
	regexOnce        sync.Once
)

func initRegex() {
	createTableRegex = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\(([\s\S]*?)\);`)
}

type SchemaParser struct {
	Config *config.Config
}

func NewSchemaParser(cfg *config.Config) *SchemaParser {
	regexOnce.Do(initRegex)
	return &SchemaParser{Config: cfg}
}

func (p *SchemaParser) Parse() (*Schema, error) {
	schema := &Schema{
		Tables: []*Table{},
		Enums:  []*Enum{},
	}

	schemaPath := p.Config.SchemaPath
	if !filepath.IsAbs(schemaPath) {
		cwd, _ := os.Getwd()
		schemaPath = filepath.Join(cwd, schemaPath)
	}

	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		schemaDir := filepath.Dir(schemaPath)
		files, err := filepath.Glob(filepath.Join(schemaDir, "*.sql"))
		if err != nil || len(files) == 0 {
			return schema, nil
		}

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				continue
			}

			if err := utils.ValidateSchemaSyntax(string(content), file); err != nil {
				return nil, err
			}

			tables := p.parseCreateTables(string(content))
			schema.Tables = append(schema.Tables, tables...)
			enums := p.parseCreateEnums(string(content))
			schema.Enums = append(schema.Enums, enums...)
		}
	} else {
		content, err := os.ReadFile(schemaPath)
		if err != nil {
			return schema, nil
		}

		if err := utils.ValidateSchemaSyntax(string(content), schemaPath); err != nil {
			return nil, err
		}

		tables := p.parseCreateTables(string(content))
		schema.Tables = append(schema.Tables, tables...)
		enums := p.parseCreateEnums(string(content))
		schema.Enums = append(schema.Enums, enums...)
	}

	return schema, nil
}

func (p *SchemaParser) parseCreateTables(sql string) []*Table {
	sql = utils.RemoveComments(sql)

	tables := make([]*Table, 0, 8)
	matches := createTableRegex.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		table := &Table{
			Name:    match[1],
			Columns: make([]*Column, 0, 16),
		}

		lines := utils.SplitColumns(match[2])
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			lineUpper := strings.ToUpper(line)
			if strings.HasPrefix(lineUpper, "PRIMARY") ||
				strings.HasPrefix(lineUpper, "FOREIGN") ||
				strings.HasPrefix(lineUpper, "UNIQUE") ||
				strings.HasPrefix(lineUpper, "CHECK") ||
				strings.HasPrefix(lineUpper, "CONSTRAINT") ||
				strings.HasPrefix(lineUpper, "INDEX") ||
				strings.HasPrefix(lineUpper, "KEY") {
				continue
			}

			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			isNullable := !strings.Contains(lineUpper, "NOT NULL") &&
				!strings.Contains(lineUpper, "PRIMARY KEY") &&
				!strings.Contains(strings.ToUpper(parts[1]), "SERIAL")

			table.Columns = append(table.Columns, &Column{
				Name:     parts[0],
				Type:     parts[1],
				Nullable: isNullable,
			})
		}

		if len(table.Columns) > 0 {
			tables = append(tables, table)
		}
	}

	return tables
}

func (p *SchemaParser) parseCreateEnums(sql string) []*Enum {
	sql = utils.RemoveComments(sql)

	enums := make([]*Enum, 0)
	enumRegex := regexp.MustCompile(`(?i)CREATE\s+TYPE\s+(\w+)\s+AS\s+ENUM\s*\(\s*([^)]+)\s*\)`)
	matches := enumRegex.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		enumName := match[1]
		valuesStr := match[2]

		var values []string
		for _, v := range strings.Split(valuesStr, ",") {
			v = strings.TrimSpace(v)
			v = strings.Trim(v, "'\"")
			if v != "" {
				values = append(values, v)
			}
		}

		if len(values) > 0 {
			enums = append(enums, &Enum{
				Name:   enumName,
				Values: values,
			})
		}
	}

	return enums
}
