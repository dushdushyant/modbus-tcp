# Modbus TCP Adaptor

This Go application reads registers from Modbus TCP devices using per-register and common configuration files. It supports:
- Per-register config: slave id, modbus address, data type, size
- Common config: polling frequency, write-to-file, output file, rolling log, MQTT enable, etc.
- Optional file output and MQTT posting

## Usage
1. Place register config files in the `configs/` directory.
2. Edit the common config file (`common_config.yaml`).
3. Run the application:
   ```sh
   go run main.go
   ```

## Dependencies
- github.com/goburrow/modbus
- github.com/eclipse/paho.mqtt.golang
- gopkg.in/natefinch/lumberjack.v2

## To Do
- Implement config loading
- Implement Modbus polling
- Add file and MQTT output
- Add rolling log
