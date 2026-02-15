package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	wslctl_server "github.com/wslctl/wslctl-server/wslctl-server"
	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"gopkg.in/yaml.v3"
)

func main() {
	service := wslctl_server.NewService("wslctl-server", "WSL Control Server")

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "register":
			if err := service.Register(); err != nil {
				log.Fatalf("Register failed: %v", err)
			}
			return

		case "unregister":
			if err := service.Unregister(); err != nil {
				log.Fatalf("Unregister failed: %v", err)
			}
			return

		case "start":
			if err := service.Start(); err != nil {
				log.Fatalf("Start failed: %v", err)
			}
			return
		case "stop":
			if err := service.Stop(); err != nil {
				log.Fatalf("Stop failed: %v", err)
			}
			return
		case "test":
			schemaFile, err := os.Open("./schemas/drafts/2026.draft.schema.yaml")
			if err != nil {
				log.Fatalf("Error opening YAML file: %v", err)
			}
			defer schemaFile.Close()

			var schemaData any
			decoder := yaml.NewDecoder(schemaFile)
			err = decoder.Decode(&schemaData)
			if err != nil {
				log.Fatalf("Error decoding YAML file: %v", err)
			}

			jsonSchema, err := json.Marshal(schemaData)
			if err != nil {
				log.Fatalf("Error converting YAML to JSON: %v", err)
			}

			schemaLoader := gojsonschema.NewStringLoader(string(jsonSchema))
			schema, err := gojsonschema.NewSchema(schemaLoader)
			if err != nil {
				log.Fatalf("Error loading YAML schema: %v", err)
			}

			testFile, err := os.Open("./tests/test.yaml")
			if err != nil {
				log.Fatalf("Error dopening YAML file: %v", err)
			}
			defer testFile.Close()

			var testData any
			decoder = yaml.NewDecoder(testFile)
			err = decoder.Decode(&testData)
			if err != nil {
				log.Fatalf("Error decoding YAML: %v", err)
			}

			jsonTestData, err := json.Marshal(testData)
			if err != nil {
				log.Fatalf("Error converting YAML to JSON: %v", err)
			}

			jsonTestLoader := gojsonschema.NewStringLoader(string(jsonTestData))
			result, err := schema.Validate(jsonTestLoader)
			if err != nil {
				log.Fatalf("YAML is invalid according to the schema: %v", err)
			}

			fmt.Printf("YAML is valid according to the schema: %s\n", strconv.FormatBool(result.Valid()))

			var distros wslctl_server.DistributionsList
			err = json.Unmarshal(jsonTestData, &distros)

			if len(distros.Distributions) > 0 {
				fmt.Printf("%v\n", distros)
			}

			return
		}
	}

	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to check if running as a service: %v", err)
	}

	if isService {
		// Running as a Windows service
		err = svc.Run(service.GetName(), service)
		if err != nil {
			log.Fatalf("Failed to run as service: %v", err)
		}
	} else {
		// Running in debug mode
		fmt.Println("Running in debug mode (Ctrl+C to exit)")
		elog, err := eventlog.Open(service.GetName())
		if err != nil {
			log.Fatalf("Failed to open event log: %v", err)
		}
		defer elog.Close()

		// Start pipe server even in debug mode
		go service.RunPipeServer(&eventlog.Log{})

		// Simulate service running in debug mode
		for {
			elog.Info(1, "hello (debug)")
			time.Sleep(1 * time.Minute)
		}
	}
}
