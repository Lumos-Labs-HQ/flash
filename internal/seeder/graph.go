package seeder

import "fmt"

type DependencyGraph struct {
	tables map[string]*TableInfo
	order  []string
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		tables: make(map[string]*TableInfo),
	}
}

func (g *DependencyGraph) AddTable(table *TableInfo) {
	g.tables[table.Name] = table
}

func (g *DependencyGraph) BuildInsertionOrder() ([]string, error) {
	visited := make(map[string]bool)
	temp := make(map[string]bool)
	var order []string

	var visit func(string) error
	visit = func(tableName string) error {
		if temp[tableName] {
			return fmt.Errorf("circular dependency detected involving table: %s", tableName)
		}
		if visited[tableName] {
			return nil
		}

		temp[tableName] = true
		table := g.tables[tableName]
		
		if table != nil {
			for _, dep := range table.Dependencies {
				if dep != tableName { // Skip self-references
					if err := visit(dep); err != nil {
						return err
					}
				}
			}
		}

		temp[tableName] = false
		visited[tableName] = true
		order = append(order, tableName)
		return nil
	}

	for tableName := range g.tables {
		if !visited[tableName] {
			if err := visit(tableName); err != nil {
				return nil, err
			}
		}
	}

	g.order = order
	return order, nil
}

func (g *DependencyGraph) GetOrder() []string {
	return g.order
}
