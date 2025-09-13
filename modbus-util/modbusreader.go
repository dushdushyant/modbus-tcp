package modbusutil

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/goburrow/modbus"
)

// parseRegisterValue decodes Modbus register bytes based on the sensor's DataType.
func parseRegisterValue(data []byte, dataType string) interface{} {
	// Helper for BCD decoding
	bcdToUint := func(b []byte) uint64 {
		var val uint64
		for _, d := range b {
			hi := (d >> 4) & 0x0F
			lo := d & 0x0F
			val = val*100 + uint64(hi)*10 + uint64(lo)
		}
		return val
	}
	switch dataType {
	case "hex":
		return fmt.Sprintf("%X", data)
	case "bcd":
		return bcdToUint(data)
	case "uint8":
		if len(data) >= 1 {
			return data[0]
		}
	case "int8":
		if len(data) >= 1 {
			return int8(data[0])
		}
	case "uint16":
		if len(data) >= 2 {
			return binary.BigEndian.Uint16(data)
		}
	case "int16":
		if len(data) >= 2 {
			return int16(binary.BigEndian.Uint16(data))
		}
	case "uint32":
		if len(data) >= 4 {
			return binary.BigEndian.Uint32(data)
		}
	case "int32":
		if len(data) >= 4 {
			return int32(binary.BigEndian.Uint32(data))
		}
	case "uint64":
		if len(data) >= 8 {
			return binary.BigEndian.Uint64(data)
		}
	case "int64":
		if len(data) >= 8 {
			return int64(binary.BigEndian.Uint64(data))
		}
	case "float32":
		if len(data) >= 4 {
			bits := binary.BigEndian.Uint32(data)
			return math.Float32frombits(bits)
		}
	case "float64":
		if len(data) >= 8 {
			bits := binary.BigEndian.Uint64(data)
			return math.Float64frombits(bits)
		}
	case "bool":
		if len(data) >= 1 {
			return data[0] != 0
		}
	case "string":
		return string(data)
	}
	return data // fallback: return raw bytes
}

type Sensor struct {
	RegisterName  string `json:"register_name"`
	SlaveID       byte   `json:"slave_id"`
	ModbusAddress uint16 `json:"modbus_address"`
	DataType      string `json:"data_type"`
	Size          uint16 `json:"size"`
	IP            string `json:"ip"`
}

type ModbusConfig struct {
	PollingInterval int      `json:"polling_interval"`
	Timeout         int      `json:"timeout"`
	Sensors         []Sensor `json:"sensors"`
}

func ReadSensors(config ModbusConfig, interval time.Duration, timeout time.Duration, out chan<- []byte, stop <-chan struct{}, logger *log.Logger) {
	for {
		select {
		case <-stop:
			logger.Println("Stopping Modbus reader goroutine.")
			return
		default:
			for _, sensor := range config.Sensors {
				handler := modbus.NewTCPClientHandler(sensor.IP + ":502")
				handler.SlaveId = sensor.SlaveID
				handler.Timeout = timeout
				err := handler.Connect()
				if err != nil {
					logger.Printf("Modbus connection error for %s: %v", sensor.RegisterName, err)
					msg := map[string]interface{}{
						"register": sensor.RegisterName,
						"value":    "abc",
						"ts":       time.Now().UTC(),
					}
					payload, err := json.Marshal(msg)
					if err == nil {
						out <- payload
					}
					continue
				}
				client := modbus.NewClient(handler)
				results, err := client.ReadHoldingRegisters(sensor.ModbusAddress, sensor.Size)
				handler.Close()
				if err != nil {
					logger.Printf("Modbus read error for %s: %v", sensor.RegisterName, err)
					continue
				}
				msg := map[string]interface{}{
					"register":       sensor.RegisterName,
					"value":          parseRegisterValue(results, sensor.DataType),
					"starttimestamp": time.Now().UTC(),
				}
				payload, err := json.Marshal(msg)
				if err == nil {
					out <- payload
				} else {
					logger.Printf("JSON marshal error for %s: %v", sensor.RegisterName, err)
				}
			}
			time.Sleep(interval)
		}
	}
}
