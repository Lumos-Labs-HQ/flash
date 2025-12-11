package seeder

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

//go:embed data.json
var dataJSON []byte

type FakeData struct {
	FirstNames []string `json:"firstNames"`
	LastNames  []string `json:"lastNames"`
	Domains    []string `json:"domains"`
	Titles     []string `json:"titles"`
	Sentences  []string `json:"sentences"`
	Paragraphs []string `json:"paragraphs"`
	Cities     []string `json:"cities"`
	States     []string `json:"states"`
	Streets    []string `json:"streets"`
	Companies  []string `json:"companies"`
	Products   []string `json:"products"`
	Tags       []string `json:"tags"`
	Statuses   []string `json:"statuses"`
	Categories []string `json:"categories"`
}

type DataGenerator struct {
	rand     *rand.Rand
	counter  int
	fakeData *FakeData
}

func NewDataGenerator() *DataGenerator {
	var data FakeData
	if err := json.Unmarshal(dataJSON, &data); err != nil {
		// Fallback to empty data
		data = FakeData{}
	}

	return &DataGenerator{
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		counter:  0,
		fakeData: &data,
	}
}

func (g *DataGenerator) GenerateForColumn(colName, colType string, nullable bool) interface{} {
	if nullable && g.rand.Intn(10) < 2 {
		return nil
	}

	colLower := strings.ToLower(colName)
	
	// Context-aware generation based on column name
	if strings.Contains(colLower, "email") {
		return g.generateEmail()
	}
	if strings.Contains(colLower, "first") && strings.Contains(colLower, "name") {
		return g.generateFirstName()
	}
	if strings.Contains(colLower, "last") && strings.Contains(colLower, "name") {
		return g.generateLastName()
	}
	if strings.Contains(colLower, "name") && !strings.Contains(colLower, "file") && !strings.Contains(colLower, "user") {
		return g.generateName()
	}
	if strings.Contains(colLower, "title") {
		return g.generateTitle()
	}
	if strings.Contains(colLower, "description") {
		return g.generateParagraph()
	}
	if strings.Contains(colLower, "content") || strings.Contains(colLower, "body") {
		return g.generateParagraph()
	}
	if strings.Contains(colLower, "url") || strings.Contains(colLower, "link") || strings.Contains(colLower, "website") {
		return g.generateURL()
	}
	if strings.Contains(colLower, "phone") {
		return g.generatePhone()
	}
	if strings.Contains(colLower, "address") {
		return g.generateAddress()
	}
	if strings.Contains(colLower, "city") {
		return g.generateCity()
	}
	if strings.Contains(colLower, "state") {
		return g.generateState()
	}
	if strings.Contains(colLower, "zip") || strings.Contains(colLower, "postal") {
		return g.generateZipCode()
	}
	if strings.Contains(colLower, "company") || strings.Contains(colLower, "organization") {
		return g.generateCompany()
	}
	if strings.Contains(colLower, "product") {
		return g.generateProduct()
	}
	if strings.Contains(colLower, "tag") {
		return g.generateTag()
	}
	if strings.Contains(colLower, "status") {
		return g.generateStatus()
	}
	if strings.Contains(colLower, "category") {
		return g.generateCategory()
	}
	if strings.Contains(colLower, "price") || strings.Contains(colLower, "amount") || strings.Contains(colLower, "cost") {
		return g.generatePrice()
	}
	if strings.Contains(colLower, "quantity") || strings.Contains(colLower, "count") {
		return g.rand.Intn(100) + 1
	}
	if strings.Contains(colLower, "rating") || strings.Contains(colLower, "score") {
		return g.rand.Intn(5) + 1
	}

	// Fall back to type-based generation
	return g.Generate(colType, nullable)
}

func (g *DataGenerator) Generate(colType string, nullable bool) interface{} {
	if nullable && g.rand.Intn(10) < 2 {
		return nil
	}

	typeUpper := strings.ToUpper(colType)
	if idx := strings.Index(typeUpper, "("); idx > 0 {
		typeUpper = typeUpper[:idx]
	}

	switch {
	case strings.Contains(typeUpper, "INT") || strings.Contains(typeUpper, "SERIAL"):
		return g.rand.Intn(1000000) + 1
	case strings.Contains(typeUpper, "VARCHAR") || strings.Contains(typeUpper, "TEXT"):
		return g.generateSentence()
	case strings.Contains(typeUpper, "BOOL"):
		return g.rand.Intn(2) == 1
	case strings.Contains(typeUpper, "TIMESTAMP") || strings.Contains(typeUpper, "DATETIME"):
		return g.generateTimestamp()
	case strings.Contains(typeUpper, "DATE"):
		return g.generateDate()
	case strings.Contains(typeUpper, "DECIMAL") || strings.Contains(typeUpper, "NUMERIC") || strings.Contains(typeUpper, "FLOAT") || strings.Contains(typeUpper, "DOUBLE"):
		return g.generatePrice()
	case strings.Contains(typeUpper, "UUID"):
		return g.generateUUID()
	case strings.Contains(typeUpper, "JSON"):
		return `{"generated": true}`
	default:
		return g.generateSentence()
	}
}

func (g *DataGenerator) generateFirstName() string {
	if len(g.fakeData.FirstNames) == 0 {
		return "John"
	}
	return g.fakeData.FirstNames[g.rand.Intn(len(g.fakeData.FirstNames))]
}

func (g *DataGenerator) generateLastName() string {
	if len(g.fakeData.LastNames) == 0 {
		return "Doe"
	}
	return g.fakeData.LastNames[g.rand.Intn(len(g.fakeData.LastNames))]
}

func (g *DataGenerator) generateName() string {
	return g.generateFirstName() + " " + g.generateLastName()
}

func (g *DataGenerator) generateEmail() string {
	g.counter++
	if len(g.fakeData.Domains) == 0 {
		return fmt.Sprintf("user%d@example.com", g.counter)
	}
	firstName := strings.ToLower(g.generateFirstName())
	lastName := strings.ToLower(g.generateLastName())
	domain := g.fakeData.Domains[g.rand.Intn(len(g.fakeData.Domains))]
	return fmt.Sprintf("%s.%s%d@%s", firstName, lastName, g.counter, domain)
}

func (g *DataGenerator) generateTitle() string {
	if len(g.fakeData.Titles) == 0 {
		return "Sample Title"
	}
	return g.fakeData.Titles[g.rand.Intn(len(g.fakeData.Titles))]
}

func (g *DataGenerator) generateSentence() string {
	if len(g.fakeData.Sentences) == 0 {
		return "This is a sample text."
	}
	return g.fakeData.Sentences[g.rand.Intn(len(g.fakeData.Sentences))]
}

func (g *DataGenerator) generateParagraph() string {
	if len(g.fakeData.Paragraphs) == 0 {
		return "This is a sample paragraph with some content."
	}
	return g.fakeData.Paragraphs[g.rand.Intn(len(g.fakeData.Paragraphs))]
}

func (g *DataGenerator) generateCity() string {
	if len(g.fakeData.Cities) == 0 {
		return "New York"
	}
	return g.fakeData.Cities[g.rand.Intn(len(g.fakeData.Cities))]
}

func (g *DataGenerator) generateState() string {
	if len(g.fakeData.States) == 0 {
		return "California"
	}
	return g.fakeData.States[g.rand.Intn(len(g.fakeData.States))]
}

func (g *DataGenerator) generateStreet() string {
	if len(g.fakeData.Streets) == 0 {
		return "Main Street"
	}
	return g.fakeData.Streets[g.rand.Intn(len(g.fakeData.Streets))]
}

func (g *DataGenerator) generateAddress() string {
	number := g.rand.Intn(9999) + 1
	street := g.generateStreet()
	city := g.generateCity()
	state := g.generateState()
	zip := g.generateZipCode()
	return fmt.Sprintf("%d %s, %s, %s %s", number, street, city, state, zip)
}

func (g *DataGenerator) generateZipCode() string {
	return fmt.Sprintf("%05d", g.rand.Intn(100000))
}

func (g *DataGenerator) generateCompany() string {
	if len(g.fakeData.Companies) == 0 {
		return "Tech Company Inc"
	}
	return g.fakeData.Companies[g.rand.Intn(len(g.fakeData.Companies))]
}

func (g *DataGenerator) generateProduct() string {
	if len(g.fakeData.Products) == 0 {
		return "Product"
	}
	return g.fakeData.Products[g.rand.Intn(len(g.fakeData.Products))]
}

func (g *DataGenerator) generateTag() string {
	if len(g.fakeData.Tags) == 0 {
		return "tag"
	}
	return g.fakeData.Tags[g.rand.Intn(len(g.fakeData.Tags))]
}

func (g *DataGenerator) generateStatus() string {
	if len(g.fakeData.Statuses) == 0 {
		return "active"
	}
	return g.fakeData.Statuses[g.rand.Intn(len(g.fakeData.Statuses))]
}

func (g *DataGenerator) generateCategory() string {
	if len(g.fakeData.Categories) == 0 {
		return "General"
	}
	return g.fakeData.Categories[g.rand.Intn(len(g.fakeData.Categories))]
}

func (g *DataGenerator) generateURL() string {
	return fmt.Sprintf("https://%s.com/%s", strings.ToLower(g.generateLastName()), strings.ToLower(g.generateFirstName()))
}

func (g *DataGenerator) generatePhone() string {
	return fmt.Sprintf("+1-%03d-%03d-%04d", g.rand.Intn(900)+100, g.rand.Intn(900)+100, g.rand.Intn(10000))
}

func (g *DataGenerator) generatePrice() float64 {
	return float64(g.rand.Intn(100000)) / 100.0
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
