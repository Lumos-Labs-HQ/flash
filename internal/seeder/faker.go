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
	FirstNames   []string `json:"firstNames"`
	LastNames    []string `json:"lastNames"`
	Domains      []string `json:"domains"`
	ImageUrls    []string `json:"imageUrls"`
	AvatarUrls   []string `json:"avatarUrls"`
	DocumentUrls []string `json:"documentUrls"`
	Titles       []string `json:"titles"`
	Sentences    []string `json:"sentences"`
	Paragraphs   []string `json:"paragraphs"`
	Cities       []string `json:"cities"`
	States       []string `json:"states"`
	Streets      []string `json:"streets"`
	Companies    []string `json:"companies"`
	Products     []string `json:"products"`
	Tags         []string `json:"tags"`
	Statuses     []string `json:"statuses"`
	Categories   []string `json:"categories"`
}

type DataGenerator struct {
	rand     *rand.Rand
	counter  int
	fakeData *FakeData
	patterns map[string]func() interface{}
}

func NewDataGenerator() *DataGenerator {
	var data FakeData
	json.Unmarshal(dataJSON, &data)

	g := &DataGenerator{
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
		fakeData: &data,
	}
	g.initPatterns()
	return g
}

func (g *DataGenerator) initPatterns() {
	g.patterns = map[string]func() interface{}{
		// Security - Documents always NULL
		"document|doc|file|attachment|pdf|upload": func() interface{} { return nil },
		
		// Media
		"image|img|photo|picture|thumbnail|banner": g.randomFrom(g.fakeData.ImageUrls, "https://picsum.photos/400/300"),
		"avatar|profile_pic|profile_image":         g.randomFrom(g.fakeData.AvatarUrls, "https://i.pravatar.cc/150"),
		
		// Identity
		"first_name|firstname": g.randomFrom(g.fakeData.FirstNames, "John"),
		"last_name|lastname":   g.randomFrom(g.fakeData.LastNames, "Doe"),
		"email":                g.generateEmail,
		"name":                 g.generateFullName,
		
		// Content
		"title":              g.randomFrom(g.fakeData.Titles, "Sample Title"),
		"description":        g.randomFrom(g.fakeData.Paragraphs, "Sample description"),
		"content|body":       g.randomFrom(g.fakeData.Paragraphs, "Sample content"),
		"phone":              g.generatePhone,
		"url|link|website":   g.generateURL,
		
		// Location
		"address":     g.generateAddress,
		"city":        g.randomFrom(g.fakeData.Cities, "New York"),
		"state":       g.randomFrom(g.fakeData.States, "California"),
		"zip|postal":  g.generateZip,
		
		// Business
		"company|organization": g.randomFrom(g.fakeData.Companies, "Tech Company Inc"),
		"product":              g.randomFrom(g.fakeData.Products, "Product"),
		"tag":                  g.randomFrom(g.fakeData.Tags, "tag"),
		"status":               g.randomFrom(g.fakeData.Statuses, "active"),
		"category":             g.randomFrom(g.fakeData.Categories, "General"),
		
		// Numeric
		"price|amount|cost":    func() interface{} { return float64(g.rand.Intn(100000)) / 100.0 },
		"quantity|count":       func() interface{} { return g.rand.Intn(100) + 1 },
		"rating|score":         func() interface{} { return g.rand.Intn(5) + 1 },
	}
}

func (g *DataGenerator) GenerateForColumn(colName, colType string, nullable bool) interface{} {
	if nullable && g.rand.Intn(10) < 2 {
		return nil
	}

	colLower := strings.ToLower(colName)
	for pattern, generator := range g.patterns {
		for _, keyword := range strings.Split(pattern, "|") {
			if strings.Contains(colLower, keyword) {
				return generator()
			}
		}
	}

	return g.Generate(colType, nullable)
}

func (g *DataGenerator) Generate(colType string, nullable bool) interface{} {
	if nullable && g.rand.Intn(10) < 2 {
		return nil
	}

	typeUpper := strings.ToUpper(strings.Split(colType, "(")[0])
	
	typeMap := map[string]func() interface{}{
		"INT":       func() interface{} { return g.rand.Intn(1000000) + 1 },
		"SERIAL":    func() interface{} { return g.rand.Intn(1000000) + 1 },
		"VARCHAR":   func() interface{} { return g.randomSentence() },
		"TEXT":      func() interface{} { return g.randomSentence() },
		"BOOL":      func() interface{} { return g.rand.Intn(2) == 1 },
		"TIMESTAMP": func() interface{} { return time.Now().AddDate(0, 0, -g.rand.Intn(365)) },
		"DATETIME":  func() interface{} { return time.Now().AddDate(0, 0, -g.rand.Intn(365)) },
		"DATE":      func() interface{} { return time.Now().AddDate(0, 0, -g.rand.Intn(365)).Format("2006-01-02") },
		"DECIMAL":   func() interface{} { return float64(g.rand.Intn(100000)) / 100.0 },
		"FLOAT":     func() interface{} { return float64(g.rand.Intn(100000)) / 100.0 },
		"UUID":      func() interface{} { return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", g.rand.Uint32(), g.rand.Uint32()&0xffff, g.rand.Uint32()&0xffff, g.rand.Uint32()&0xffff, g.rand.Uint64()&0xffffffffffff) },
		"JSON":      func() interface{} { return `{"generated": true}` },
	}

	for t, gen := range typeMap {
		if strings.Contains(typeUpper, t) {
			return gen()
		}
	}
	
	return g.randomSentence()
}

// Helper functions
func (g *DataGenerator) randomFrom(slice []string, fallback string) func() interface{} {
	return func() interface{} {
		if len(slice) == 0 {
			return fallback
		}
		return slice[g.rand.Intn(len(slice))]
	}
}

func (g *DataGenerator) randomSentence() string {
	if len(g.fakeData.Sentences) == 0 {
		return "Sample text"
	}
	return g.fakeData.Sentences[g.rand.Intn(len(g.fakeData.Sentences))]
}

func (g *DataGenerator) generateEmail() interface{} {
	g.counter++
	if len(g.fakeData.FirstNames) == 0 || len(g.fakeData.LastNames) == 0 || len(g.fakeData.Domains) == 0 {
		return fmt.Sprintf("user%d@example.com", g.counter)
	}
	first := strings.ToLower(g.fakeData.FirstNames[g.rand.Intn(len(g.fakeData.FirstNames))])
	last := strings.ToLower(g.fakeData.LastNames[g.rand.Intn(len(g.fakeData.LastNames))])
	domain := g.fakeData.Domains[g.rand.Intn(len(g.fakeData.Domains))]
	return fmt.Sprintf("%s.%s%d@%s", first, last, g.counter, domain)
}

func (g *DataGenerator) generateFullName() interface{} {
	first := g.randomFrom(g.fakeData.FirstNames, "John")().(string)
	last := g.randomFrom(g.fakeData.LastNames, "Doe")().(string)
	return fmt.Sprintf("%s %s", first, last)
}

func (g *DataGenerator) generatePhone() interface{} {
	return fmt.Sprintf("+1-%03d-%03d-%04d", g.rand.Intn(900)+100, g.rand.Intn(900)+100, g.rand.Intn(10000))
}

func (g *DataGenerator) generateURL() interface{} {
	first := strings.ToLower(g.randomFrom(g.fakeData.FirstNames, "john")().(string))
	last := strings.ToLower(g.randomFrom(g.fakeData.LastNames, "doe")().(string))
	return fmt.Sprintf("https://%s.com/%s", last, first)
}

func (g *DataGenerator) generateAddress() interface{} {
	street := g.randomFrom(g.fakeData.Streets, "Main Street")().(string)
	city := g.randomFrom(g.fakeData.Cities, "New York")().(string)
	state := g.randomFrom(g.fakeData.States, "NY")().(string)
	return fmt.Sprintf("%d %s, %s, %s %05d", g.rand.Intn(9999)+1, street, city, state, g.rand.Intn(100000))
}

func (g *DataGenerator) generateZip() interface{} {
	return fmt.Sprintf("%05d", g.rand.Intn(100000))
}
