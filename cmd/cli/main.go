package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"go.uber.org/zap"

	"github.com/emirozbir/micro-sre/internal/agent"
	"github.com/emirozbir/micro-sre/internal/config"
	"github.com/emirozbir/micro-sre/internal/formatter"
)

func main() {
	namespace := flag.String("namespace", "", "Kubernetes namespace")
	pod := flag.String("pod", "", "Pod name")
	lookback := flag.String("lookback", "1h", "Time range to look back (e.g., 1h, 30m)")
	configPath := flag.String("config", "", "Path to config file")
	outputFormat := flag.String("format", "pretty", "Output format: 'pretty' or 'json'")
	noColor := flag.Bool("no-color", false, "Disable colored output")

	flag.Parse()

	if *namespace == "" || *pod == "" {
		log.Fatal("Both -namespace and -pod flags are required")
	}

	// Parse lookback duration
	lookbackDuration, err := time.ParseDuration(*lookback)
	if err != nil {
		log.Fatalf("Invalid lookback duration: %v", err)
	}

	// Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize agent
	agentInstance, err := agent.NewAgent(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create agent", zap.Error(err))
	}

	// Run analysis
	fmt.Printf("🔍 Analyzing pod %s/%s (lookback: %s)...\n", *namespace, *pod, *lookback)

	ctx := context.Background()
	result, err := agentInstance.AnalyzeAlert(ctx, agent.AnalysisRequest{
		Namespace: *namespace,
		PodName:   *pod,
		Lookback:  lookbackDuration,
	})

	if err != nil {
		logger.Fatal("Analysis failed", zap.Error(err))
	}

	// Output result
	if *outputFormat == "json" {
		// JSON output
		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			logger.Fatal("Failed to marshal result", zap.Error(err))
		}
		fmt.Println(string(output))
	} else {
		// Pretty formatted output
		outputFormatter := formatter.NewFormatter(!*noColor)
		formattedOutput := outputFormatter.FormatAnalysisResult(result)
		fmt.Println(formattedOutput)
	}
}
