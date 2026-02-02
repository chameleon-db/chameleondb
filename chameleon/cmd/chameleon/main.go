package main

import (
	"fmt"
	"os"

	"github.com/dperalta86/chameleondb/chameleon/pkg/engine"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version":
		cmdVersion()
	case "parse":
		if len(os.Args) < 3 {
			fmt.Println("Usage: chameleon parse <schema-file>")
			os.Exit(1)
		}
		cmdParse(os.Args[2])
	case "validate":
		if len(os.Args) < 3 {
			fmt.Println("Usage: chameleon validate <schema-file>")
			os.Exit(1)
		}
		cmdValidate(os.Args[2])
	case "query":
		if len(os.Args) < 3 {
			fmt.Println("Usage: chameleon query <schema.cham>")
			fmt.Println("       Runs an interactive query session")
			os.Exit(1)
		}
		schemaFile := os.Args[2]
		runQuerySession(schemaFile)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("ChameleonDB - Graph-oriented database access language")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  chameleon version              Show version")
	fmt.Println("  chameleon parse <file>         Parse and display schema")
	fmt.Println("  chameleon validate <file>      Validate schema")
}

func cmdVersion() {
	eng := engine.NewEngine()
	fmt.Printf("ChameleonDB v%s\n", eng.Version())
}

func cmdParse(filepath string) {
	eng := engine.NewEngine()

	schema, err := eng.LoadSchemaFromFile(filepath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	json, err := schema.ToJSON()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(json)
}

func cmdValidate(filepath string) {
	eng := engine.NewEngine()

	_, err := eng.LoadSchemaFromFile(filepath)
	if err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Schema is valid")
}

func runQuerySession(schemaFile string) {
	eng := engine.NewEngine()

	_, err := eng.LoadSchemaFromFile(schemaFile)
	if err != nil {
		fmt.Printf("Error loading schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("ChameleonDB Query Session")
	fmt.Println("========================")
	fmt.Println("Schema loaded. Generating sample queries...")

	// Demo: show what queries would generate
	demos := []struct {
		name  string
		query func() (*engine.GeneratedSQL, error)
	}{
		{
			"Fetch all users",
			func() (*engine.GeneratedSQL, error) {
				return eng.Query("User").ToSQL()
			},
		},
		{
			"Filter by email",
			func() (*engine.GeneratedSQL, error) {
				return eng.Query("User").
					Filter("email", "eq", "ana@mail.com").
					ToSQL()
			},
		},
		{
			"Users with orders > 100",
			func() (*engine.GeneratedSQL, error) {
				return eng.Query("User").
					Filter("orders.total", "gt", 100).
					Include("orders").
					ToSQL()
			},
		},
		{
			"Full query",
			func() (*engine.GeneratedSQL, error) {
				return eng.Query("User").
					Filter("age", "gte", 18).
					Filter("orders.total", "gt", 50).
					Include("orders").
					Include("orders.items").
					OrderBy("name", "desc").
					Limit(10).
					ToSQL()
			},
		},
	}

	for _, demo := range demos {
		fmt.Printf("── %s ──\n", demo.name)
		result, err := demo.query()
		if err != nil {
			fmt.Printf("  Error: %v\n\n", err)
			continue
		}

		fmt.Printf("  Main:\n    %s\n", result.MainQuery)
		for _, eq := range result.EagerQueries {
			fmt.Printf("  Eager (%s):\n    %s\n", eq[0], eq[1])
		}
		fmt.Println()
	}
}
