package cmd

import (
	"fmt"
	"os"

	"github.com/Rana718/Graft/internal/config"
	"github.com/sqlc-dev/sqlc/pkg/cli"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate SQLC types",
	Long: `
Generate Go types from SQL queries using embedded SQLC.
This command uses built-in SQLC configuration from graft.config.json.

Requirements:
- Configuration in graft.config.json
- Valid SQL schema and queries

This command will:
1. Generate temporary SQLC config from graft.config.json
2. Run embedded SQLC generate to update Go types
3. Clean up temporary files`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if err := runSQLCGenerate(cfg); err != nil {
			return fmt.Errorf("failed to run SQLC generate: %w", err)
		}

		fmt.Println("🎉 SQLC types generated successfully!")
		return nil
	},
}

type sqlcConfig struct {
	Version string    `yaml:"version"`
	SQL     []sqlcSQL `yaml:"sql"`
}

type sqlcSQL struct {
	Engine  string     `yaml:"engine"`
	Queries string     `yaml:"queries"`
	Schema  string     `yaml:"schema"`
	Gen     sqlcGenCfg `yaml:"gen"`
}

type sqlcGenCfg struct {
	Go sqlcGoCfg `yaml:"go"`
}

type sqlcGoCfg struct {
	Package                   string `yaml:"package"`
	Out                       string `yaml:"out"`
	SqlPackage                string `yaml:"sql_package,omitempty"`
	EmitInterface             bool   `yaml:"emit_interface,omitempty"`
	EmitJsonTags              bool   `yaml:"emit_json_tags,omitempty"`
	EmitDbTags                bool   `yaml:"emit_db_tags,omitempty"`
	EmitPreparedQueries       bool   `yaml:"emit_prepared_queries,omitempty"`
	EmitExactTableNames       bool   `yaml:"emit_exact_table_names,omitempty"`
	EmitEmptySlices           bool   `yaml:"emit_empty_slices,omitempty"`
	EmitExportedQueries       bool   `yaml:"emit_exported_queries,omitempty"`
	EmitResultStructPointers  bool   `yaml:"emit_result_struct_pointers,omitempty"`
	EmitParamsStructPointers  bool   `yaml:"emit_params_struct_pointers,omitempty"`
	EmitMethodsWithDbArgument bool   `yaml:"emit_methods_with_db_argument,omitempty"`
	EmitPointersForNullTypes  bool   `yaml:"emit_pointers_for_null_types,omitempty"`
	EmitEnumValidMethod       bool   `yaml:"emit_enum_valid_method,omitempty"`
	EmitAllEnumValues         bool   `yaml:"emit_all_enum_values,omitempty"`
	JsonTagsCaseStyle         string `yaml:"json_tags_case_style,omitempty"`
	OutputDbFileName          string `yaml:"output_db_file_name,omitempty"`
	OutputModelsFileName      string `yaml:"output_models_file_name,omitempty"`
	OutputQuerierFileName     string `yaml:"output_querier_file_name,omitempty"`
	OutputFilesSuffix         string `yaml:"output_files_suffix,omitempty"`
}

func runSQLCGenerate(cfg *config.Config) error {
	tmpFile := ".graft_sqlc_temp.yaml"
	defer os.Remove(tmpFile)

	goCfg := sqlcGoCfg{
		Package:                   "graft",
		Out:                       "graft_gen/",
		SqlPackage:                cfg.Gen.Go.SqlPackage,
		EmitInterface:             cfg.Gen.Go.EmitInterface,
		EmitJsonTags:              cfg.Gen.Go.EmitJsonTags,
		EmitDbTags:                cfg.Gen.Go.EmitDbTags,
		EmitPreparedQueries:       cfg.Gen.Go.EmitPreparedQueries,
		EmitExactTableNames:       cfg.Gen.Go.EmitExactTableNames,
		EmitEmptySlices:           cfg.Gen.Go.EmitEmptySlices,
		EmitExportedQueries:       cfg.Gen.Go.EmitExportedQueries,
		EmitResultStructPointers:  cfg.Gen.Go.EmitResultStructPointers,
		EmitParamsStructPointers:  cfg.Gen.Go.EmitParamsStructPointers,
		EmitMethodsWithDbArgument: cfg.Gen.Go.EmitMethodsWithDbArgument,
		EmitPointersForNullTypes:  cfg.Gen.Go.EmitPointersForNullTypes,
		EmitEnumValidMethod:       cfg.Gen.Go.EmitEnumValidMethod,
		EmitAllEnumValues:         cfg.Gen.Go.EmitAllEnumValues,
		JsonTagsCaseStyle:         cfg.Gen.Go.JsonTagsCaseStyle,
		OutputDbFileName:          cfg.Gen.Go.OutputDbFileName,
		OutputModelsFileName:      cfg.Gen.Go.OutputModelsFileName,
		OutputQuerierFileName:     cfg.Gen.Go.OutputQuerierFileName,
		OutputFilesSuffix:         cfg.Gen.Go.OutputFilesSuffix,
	}

	sqlcCfg := sqlcConfig{
		Version: cfg.Version,
		SQL: []sqlcSQL{
			{
				Engine:  cfg.GetSqlcEngine(),
				Queries: cfg.Queries,
				Schema:  cfg.GetSchemaDir(),
				Gen: sqlcGenCfg{
					Go: goCfg,
				},
			},
		},
	}

	data, err := yaml.Marshal(&sqlcCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal SQLC config: %w", err)
	}

	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary SQLC config: %w", err)
	}

	exitCode := cli.Run([]string{"generate", "-f", tmpFile})
	if exitCode != 0 {
		return fmt.Errorf("sqlc generate failed with exit code %d", exitCode)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(genCmd)
}
