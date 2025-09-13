package mqttutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Config struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	QoS      byte   `json:"qos"`
	Cert     string `json:"cert"`
	Topic    string `json:"topic"`
}

type Publisher struct {
	config *Config
	client mqtt.Client
	logger *log.Logger
}

func NewPublisher(cfg *Config, logger *log.Logger) (*Publisher, error) {
	opts := mqtt.NewClientOptions()
	broker := fmt.Sprintf("tcp://%s:%d", cfg.Hostname, cfg.Port)
	if cfg.Cert != "" {
		broker = fmt.Sprintf("ssl://%s:%d", cfg.Hostname, cfg.Port)
	}
	opts.AddBroker(broker)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)

	// Callback: OnConnect
	opts.OnConnect = func(c mqtt.Client) {
		logger.Println("[MQTT] Connected to broker")
	}

	// Callback: OnConnectionLost
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		logger.Printf("[MQTT] Connection lost: %v", err)
	}

	// Callback: OnReconnecting
	opts.OnReconnecting = func(c mqtt.Client, opts *mqtt.ClientOptions) {
		logger.Println("[MQTT] Reconnecting to broker...")
	}

	if cfg.Cert != "" {
		tlsConfig, err := newTLSConfig(cfg.Cert)
		if err != nil {
			return nil, err
		}
		opts.SetTLSConfig(tlsConfig)
	}

	client := mqtt.NewClient(opts)
	return &Publisher{config: cfg, client: client, logger: logger}, nil
}

func newTLSConfig(caFile string) (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}

func (p *Publisher) Connect() error {
	token := p.client.Connect()
	if token.Wait() && token.Error() != nil {
		p.logger.Printf("[MQTT] Connect error: %v", token.Error())
		return token.Error()
	}
	p.logger.Println("[MQTT] Connect successful")
	return nil
}

func (p *Publisher) Publish(payload []byte) error {
	token := p.client.Publish(p.config.Topic, p.config.QoS, false, payload)
	token.Wait()
	if token.Error() == nil {
		p.logger.Printf("[MQTT] Message published to topic: %s", p.config.Topic)
		p.logger.Printf("[MQTT] Payload: %s", string(payload))
	} else {
		p.logger.Printf("[MQTT] Publish error: %v", token.Error())
	}
	return token.Error()
}
