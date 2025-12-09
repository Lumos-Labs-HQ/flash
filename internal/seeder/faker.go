package seeder

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type DataGenerator struct {
	rand    *rand.Rand
	counter int
}

func NewDataGenerator() *DataGenerator {
	return &DataGenerator{
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
		counter: 0,
	}
}

func (g *DataGenerator) GenerateForColumn(colName, colType string, nullable bool) interface{} {
	if nullable && g.rand.Intn(10) < 2 {
		return nil
	}

	// Check column name first for context-aware generation
	colLower := strings.ToLower(colName)
	
	if strings.Contains(colLower, "email") {
		return g.generateEmail()
	}
	if strings.Contains(colLower, "name") && !strings.Contains(colLower, "file") && !strings.Contains(colLower, "user") {
		return g.generateName()
	}
	if strings.Contains(colLower, "title") {
		return g.generateTitle()
	}
	if strings.Contains(colLower, "description") || strings.Contains(colLower, "content") {
		return g.generateSentence()
	}
	if strings.Contains(colLower, "url") || strings.Contains(colLower, "link") {
		return g.generateURL()
	}
	if strings.Contains(colLower, "phone") {
		return g.generatePhone()
	}
	if strings.Contains(colLower, "address") {
		return g.generateAddress()
	}

	// Fall back to type-based generation
	return g.Generate(colType, nullable)
}

func (g *DataGenerator) Generate(colType string, nullable bool) interface{} {
	if nullable && g.rand.Intn(10) < 2 {
		return nil
	}

	typeUpper := strings.ToUpper(colType)
	
	// Extract base type (e.g., VARCHAR(255) -> VARCHAR)
	if idx := strings.Index(typeUpper, "("); idx > 0 {
		typeUpper = typeUpper[:idx]
	}

	switch {
	case strings.Contains(typeUpper, "INT") || strings.Contains(typeUpper, "SERIAL"):
		return g.rand.Intn(1000000) + 1
	case strings.Contains(typeUpper, "VARCHAR") || strings.Contains(typeUpper, "TEXT"):
		return g.generateText(colType)
	case strings.Contains(typeUpper, "BOOL"):
		return g.rand.Intn(2) == 1
	case strings.Contains(typeUpper, "TIMESTAMP") || strings.Contains(typeUpper, "DATETIME"):
		return g.generateTimestamp()
	case strings.Contains(typeUpper, "DATE"):
		return g.generateDate()
	case strings.Contains(typeUpper, "DECIMAL") || strings.Contains(typeUpper, "NUMERIC") || strings.Contains(typeUpper, "FLOAT") || strings.Contains(typeUpper, "DOUBLE"):
		return g.rand.Float64() * 10000
	case strings.Contains(typeUpper, "UUID"):
		return g.generateUUID()
	case strings.Contains(typeUpper, "JSON"):
		return `{"generated": true}`
	default:
		return g.generateText(colType)
	}
}

func (g *DataGenerator) generateText(colType string) string {
	colLower := strings.ToLower(colType)
	
	if strings.Contains(colLower, "email") {
		return g.generateEmail()
	}
	if strings.Contains(colLower, "name") {
		return g.generateName()
	}
	if strings.Contains(colLower, "title") {
		return g.generateTitle()
	}
	if strings.Contains(colLower, "description") || strings.Contains(colLower, "content") {
		return g.generateSentence()
	}
	if strings.Contains(colLower, "url") || strings.Contains(colLower, "link") {
		return g.generateURL()
	}
	if strings.Contains(colLower, "phone") {
		return g.generatePhone()
	}
	if strings.Contains(colLower, "address") {
		return g.generateAddress()
	}
	
	return g.generateWord()
}

func (g *DataGenerator) generateName() string {
	firstNames := []string{"John", "Jane", "Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez"}
	return firstNames[g.rand.Intn(len(firstNames))] + " " + lastNames[g.rand.Intn(len(lastNames))]
}

func (g *DataGenerator) generateEmail() string {
	g.counter++
	domains := []string{"example.com", "test.com", "demo.com", "mail.com"}
	return fmt.Sprintf("user%d_%d@%s", g.counter, g.rand.Intn(100000), domains[g.rand.Intn(len(domains))])
}

func (g *DataGenerator) generateTitle() string {
	titles := []string{
		"Getting Started with Go",
		"Understanding Databases",
		"Web Development Best Practices",
		"Introduction to APIs",
		"Modern Software Architecture",
		"Cloud Computing Basics",
		"Data Structures and Algorithms",
		"Machine Learning Fundamentals",
	}
	return titles[g.rand.Intn(len(titles))]
}

func (g *DataGenerator) generateSentence() string {
	sentences := []string{
		"This is a sample text generated for testing purposes.",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		"The quick brown fox jumps over the lazy dog.",
		"Software development requires careful planning and execution.",
		"Database design is crucial for application performance.",
	}
	return sentences[g.rand.Intn(len(sentences))]
}

func (g *DataGenerator) generateWord() string {
	words := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"}
	return words[g.rand.Intn(len(words))]
}

func (g *DataGenerator) generateURL() string {
	return fmt.Sprintf("https://example.com/page/%d", g.rand.Intn(1000))
}

func (g *DataGenerator) generatePhone() string {
	return fmt.Sprintf("+1-%03d-%03d-%04d", g.rand.Intn(1000), g.rand.Intn(1000), g.rand.Intn(10000))
}

func (g *DataGenerator) generateAddress() string {
	return fmt.Sprintf("%d Main Street, City, State %05d", g.rand.Intn(9999)+1, g.rand.Intn(100000))
}

func (g *DataGenerator) generateTimestamp() time.Time {
	now := time.Now()
	days := g.rand.Intn(365)
	return now.AddDate(0, 0, -days)
}

func (g *DataGenerator) generateDate() string {
	return g.generateTimestamp().Format("2006-01-02")
}

func (g *DataGenerator) generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		g.rand.Uint32(),
		g.rand.Uint32()&0xffff,
		g.rand.Uint32()&0xffff,
		g.rand.Uint32()&0xffff,
		g.rand.Uint64()&0xffffffffffff,
	)
}
