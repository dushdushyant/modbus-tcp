// Main entry point for Modbus TCP adaptor
package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	modbusutil "modbus-adaptor/modbus-util"
	mqttutil "modbus-adaptor/mqtt-util"
	"modbus-adaptor/util"
)

// Logger instance
var logger *log.Logger

type LogConfig struct {
	LogFile   string `json:"filename"`
	MaxSize   int    `json:"maxSize"`
	Backups   int    `json:"backups"`
	Compress  bool   `json:"compress"`
	LogToFile bool   `json:"logToFile"`
}

func loadLogConfig(path string) (*LogConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	// Parse file appender
	appenders := raw["appenders"].(map[string]interface{})
	fileAppender := appenders["file"].(map[string]interface{})
	cfg := &LogConfig{
		LogFile:   fileAppender["filename"].(string),
		MaxSize:   int(fileAppender["maxSize"].(float64) / 1024 / 1024),
		Backups:   int(fileAppender["backups"].(float64)),
		Compress:  fileAppender["compress"].(bool),
		LogToFile: true,
	}
	return cfg, nil
}

func loadMqttConfig(path string) (*mqttutil.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg mqttutil.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func loadModbusConfig(path string) (*modbusutil.ModbusConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg modbusutil.ModbusConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func main() {
	println("Starting Modbus TCP Adaptor...")
	// Load log config
	logCfg, err := loadLogConfig("config/log_config.json")
	if err != nil {
		log.Fatalf("Failed to load log config: %v", err)
	}
	logger = util.SetupLogger(util.LoggerConfig{
		LogFile:    logCfg.LogFile,
		MaxSizeMB:  logCfg.MaxSize,
		MaxBackups: logCfg.Backups,
		LogToFile:  logCfg.LogToFile,
	})
	logger.Println("Logger initialized.")

	// Load other configs
	mqttCfg, err := loadMqttConfig("config/mqtt_config.json")
	if err != nil {
		logger.Fatalf("Failed to load MQTT config: %v", err)
	}
	modbusCfg, err := loadModbusConfig("config/modbus_config.json")
	if err != nil {
		logger.Fatalf("Failed to load Modbus config: %v", err)
	}

	logger.Printf("MQTT Config: %s %d %s", mqttCfg.Hostname, mqttCfg.Port, mqttCfg.Topic)
	// Setup MQTT publisher
	publisher, err := mqttutil.NewPublisher(mqttCfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create MQTT publisher: %v", err)
	}
	if err := publisher.Connect(); err != nil {
		logger.Fatalf("Failed to connect to MQTT broker: %v", err)
	}

	// Buffer size for readingsCh is set to 100 to handle bursty sensor data without blocking;
	readingsCh := make(chan []byte, 100)
	stopCh := make(chan struct{})

	// Use polling_interval and timeout from modbus config
	pollingInterval := 5 * time.Second
	timeout := 2 * time.Second
	if modbusCfg.PollingInterval > 0 {
		pollingInterval = time.Duration(modbusCfg.PollingInterval) * time.Second
	}
	if modbusCfg.Timeout > 0 {
		timeout = time.Duration(modbusCfg.Timeout) * time.Second
	}

	go modbusutil.ReadSensors(*modbusCfg, pollingInterval, timeout, readingsCh, stopCh, logger)

	go func() {
		for payload := range readingsCh {
			if err := publisher.Publish(payload); err != nil {
				logger.Printf("MQTT publish error: %v", err)
			}
		}
	}()

	logger.Println("Modbus TCP Adaptor started. Press Ctrl+C to exit.")
	select {}
}
