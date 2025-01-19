package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/shahariaazam/smart-insights/internal/dbinterface"
)

func (p *PostgresProvider) GetSchema(ctx context.Context) (*dbinterface.SchemaInfo, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	schema := &dbinterface.SchemaInfo{
		Tables:    make([]dbinterface.TableInfo, 0),
		Views:     make([]dbinterface.ViewInfo, 0),
		Functions: make([]dbinterface.FunctionInfo, 0),
	}

	// Get tables and their columns
	query := `
		SELECT 
			t.table_name,
			array_agg(c.column_name ORDER BY c.ordinal_position) as columns,
			array_agg(c.data_type ORDER BY c.ordinal_position) as data_types,
			array_agg(c.is_nullable ORDER BY c.ordinal_position) as nullable,
			array_agg(c.column_default ORDER BY c.ordinal_position) as defaults,
			array_agg(c.character_maximum_length ORDER BY c.ordinal_position) as char_lengths,
			array_agg(pgd.description ORDER BY c.ordinal_position) as descriptions
		FROM information_schema.tables t
		JOIN information_schema.columns c ON c.table_name = t.table_name
		LEFT JOIN pg_catalog.pg_statio_all_tables st ON st.relname = t.table_name
		LEFT JOIN pg_catalog.pg_description pgd ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
		WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE'
		GROUP BY t.table_name
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var table dbinterface.TableInfo
		var columnNames, dataTypes, nullables, defaults []sql.NullString
		var charLengths []sql.NullInt64
		var descriptions []sql.NullString

		err := rows.Scan(
			&table.Name,
			pq.Array(&columnNames),
			pq.Array(&dataTypes),
			pq.Array(&nullables),
			pq.Array(&defaults),
			pq.Array(&charLengths),
			pq.Array(&descriptions),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan table info: %w", err)
		}

		table.Columns = make([]dbinterface.ColumnInfo, len(columnNames))
		for i := range columnNames {
			var charMaxLength *int
			if charLengths[i].Valid {
				length := int(charLengths[i].Int64)
				charMaxLength = &length
			}

			table.Columns[i] = dbinterface.ColumnInfo{
				Name:          columnNames[i].String,
				DataType:      dataTypes[i].String,
				IsNullable:    nullables[i].String == "YES",
				DefaultValue:  defaults[i].String,
				CharMaxLength: charMaxLength,
				Description:   descriptions[i].String,
			}
		}

		// Get primary keys
		primaryKeys, err := p.getTablePrimaryKeys(ctx, table.Name)
		if err != nil {
			return nil, err
		}
		table.PrimaryKey = primaryKeys

		// Get foreign keys
		foreignKeys, err := p.getTableForeignKeys(ctx, table.Name)
		if err != nil {
			return nil, err
		}
		table.ForeignKeys = foreignKeys

		// Get indexes
		indexes, err := p.getTableIndexes(ctx, table.Name)
		if err != nil {
			return nil, err
		}
		table.Indexes = indexes

		schema.Tables = append(schema.Tables, table)
	}

	// Get views
	viewQuery := `
		SELECT 
			v.table_name,
			array_agg(c.column_name ORDER BY c.ordinal_position) as columns,
			array_agg(c.data_type ORDER BY c.ordinal_position) as data_types,
			array_agg(c.is_nullable ORDER BY c.ordinal_position) as nullable,
			v.view_definition,
			pgd.description
		FROM information_schema.views v
		JOIN information_schema.columns c ON c.table_name = v.table_name
		LEFT JOIN pg_catalog.pg_statio_all_tables st ON st.relname = v.table_name
		LEFT JOIN pg_catalog.pg_description pgd ON pgd.objoid = st.relid AND pgd.objsubid = 0
		WHERE v.table_schema = 'public'
		GROUP BY v.table_name, v.view_definition, pgd.description
	`

	viewRows, err := p.db.QueryContext(ctx, viewQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get views: %w", err)
	}
	defer viewRows.Close()

	for viewRows.Next() {
		var view dbinterface.ViewInfo
		var columnNames, dataTypes, nullables []sql.NullString
		var definition, description sql.NullString

		err := viewRows.Scan(
			&view.Name,
			pq.Array(&columnNames),
			pq.Array(&dataTypes),
			pq.Array(&nullables),
			&definition,
			&description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan view info: %w", err)
		}

		view.Columns = make([]dbinterface.ColumnInfo, len(columnNames))
		for i := range columnNames {
			view.Columns[i] = dbinterface.ColumnInfo{
				Name:       columnNames[i].String,
				DataType:   dataTypes[i].String,
				IsNullable: nullables[i].String == "YES",
			}
		}
		view.Definition = definition.String
		view.Description = description.String

		schema.Views = append(schema.Views, view)
	}

	return schema, nil
}

func (p *PostgresProvider) getTablePrimaryKeys(ctx context.Context, tableName string) ([]string, error) {
	query := `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		WHERE i.indrelid = $1::regclass
		AND i.indisprimary;
	`

	rows, err := p.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary keys: %w", err)
	}
	defer rows.Close()

	var primaryKeys []string
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		primaryKeys = append(primaryKeys, columnName)
	}

	return primaryKeys, nil
}

func (p *PostgresProvider) getTableForeignKeys(ctx context.Context, tableName string) ([]dbinterface.ForeignKeyInfo, error) {
	query := `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.update_rule,
			rc.delete_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_name = $1
		AND tc.table_schema = 'public'
		ORDER BY tc.constraint_name, kcu.ordinal_position;
	`

	rows, err := p.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get foreign keys: %w", err)
	}
	defer rows.Close()

	fkMap := make(map[string]*dbinterface.ForeignKeyInfo)
	for rows.Next() {
		var (
			constraintName string
			columnName     string
			refTableName   string
			refColumnName  string
			updateRule     string
			deleteRule     string
		)

		if err := rows.Scan(
			&constraintName,
			&columnName,
			&refTableName,
			&refColumnName,
			&updateRule,
			&deleteRule,
		); err != nil {
			return nil, err
		}

		fk, exists := fkMap[constraintName]
		if !exists {
			fk = &dbinterface.ForeignKeyInfo{
				Name:           constraintName,
				ColumnNames:    make([]string, 0),
				RefTableName:   refTableName,
				RefColumnNames: make([]string, 0),
				OnUpdate:       updateRule,
				OnDelete:       deleteRule,
			}
			fkMap[constraintName] = fk
		}

		fk.ColumnNames = append(fk.ColumnNames, columnName)
		fk.RefColumnNames = append(fk.RefColumnNames, refColumnName)
	}

	result := make([]dbinterface.ForeignKeyInfo, 0, len(fkMap))
	for _, fk := range fkMap {
		result = append(result, *fk)
	}

	return result, nil
}

func (p *PostgresProvider) getTableIndexes(ctx context.Context, tableName string) ([]dbinterface.IndexInfo, error) {
	query := `
		SELECT
			i.relname AS index_name,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) AS column_names,
			ix.indisunique AS is_unique,
			am.amname AS index_type
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_am am ON i.relam = am.oid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE t.relname = $1
		AND t.relkind = 'r'
		AND ix.indisprimary = false  -- Exclude primary keys
		GROUP BY i.relname, ix.indisunique, am.amname;
	`

	rows, err := p.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexes: %w", err)
	}
	defer rows.Close()

	var indexes []dbinterface.IndexInfo
	for rows.Next() {
		var index dbinterface.IndexInfo
		var columnNames []string

		if err := rows.Scan(&index.Name, pq.Array(&columnNames), &index.IsUnique, &index.Type); err != nil {
			return nil, err
		}

		index.ColumnNames = columnNames
		indexes = append(indexes, index)
	}

	return indexes, nil
}
