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

func loadYaml(filePath string) (any, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Error opening YAML file: %v", err)
	}
	defer file.Close()

	var data any
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("Error decoding YAML file: %v", err)
	}

	return data, nil
}

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
			yamlSchema, err := loadYaml("./schemas/drafts/2026.draft.schema.yaml")
			if err != nil {
				log.Fatal(err)
			}

			jsonSchema, err := json.Marshal(yamlSchema)
			if err != nil {
				log.Fatalf("Error converting YAML to JSON: %v", err)
			}

			schemaLoader := gojsonschema.NewStringLoader(string(jsonSchema))
			schema, err := gojsonschema.NewSchema(schemaLoader)
			if err != nil {
				log.Fatalf("Error loading YAML schema: %v", err)
			}

			testData, err := loadYaml("./tests/test.yaml")
			if err != nil {
				log.Fatal(err)
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
