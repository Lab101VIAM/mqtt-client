package myMQTTClient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/data"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

// Init called upon import, registers this component with the module
func init() {
	resource.RegisterComponent(sensor.API, Model, resource.Registration[sensor.Sensor, *Config]{Constructor: newSensor})
}

// To be used for functions which are not meant to be implemented in your component
var errUnimplemented = errors.New("unimplemented")

// Your model's colon-delimited-triplet (acme:demo:mybase). acme = namespace, demo = repo-name, mybase = model name
// If you plan to upload this module to the Viam registry, "acme" must match your Viam registry namespace.
var Model = resource.NewModel("bill", "mqtt", "json")

// Maps JSON component configuration attributes.
type Config struct {
	Topic       string `json:"topic"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	QoS         int    `json:"qos"`
	QueueLength int    `json:"q_length"`
}

// Implement component configuration validation and and return implicit dependencies.
func (cfg *Config) Validate(path string) ([]string, error) {
	// Check if the topic is set
	if cfg.Topic == "" {
		return nil, fmt.Errorf("topic is required for MQTT_Client %q", path)
	}

	// Check if the host is set
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required for MQTT_Client %q", path)
	}

	// Check if the port is valid
	if cfg.Port <= 0 {
		return nil, fmt.Errorf("invalid port (should be > 0) for MQTT_Client %q", path)
	}

	// Check if qos is within a valid range (usually 0 to 2 for MQTT)
	if cfg.QoS < 0 || cfg.QoS > 2 {
		return nil, fmt.Errorf("qos must be between 0 and 2 for MQTT_Client %q", path)
	}

	return []string{}, nil
}

type MQTT_Client struct {
	resource.Named
	logger        logging.Logger
	client        mqtt.Client
	Topic         string
	Host          string
	Port          int
	QoS           byte
	messageQueue  []map[string]interface{}
	queueLength   int
	latestMessage map[string]interface{}
	mutex         sync.Mutex
}

// Sensor type constructor.
// Called upon sensor instantiation when a sensor model is added to the machine configuration
func newSensor(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	s := &MQTT_Client{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}
	if err := s.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}
	return s, nil
}

// Reconfigure reconfigures with new settings.
func (s *MQTT_Client) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	// Convert the generic resource.Config to the MQTT_Client-specific Config structure
	clientConfig, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return err
	}

	// Stop existing MQTT client if connected
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(250) // Timeout in milliseconds
	}

	// Reconfigure the MQTT_Client instance with new settings from clientConfig
	s.Topic = clientConfig.Topic
	s.Host = clientConfig.Host
	s.Port = clientConfig.Port
	s.QoS = byte(clientConfig.QoS) // Assuming qos in Config is an int and needs conversion to byte
	s.queueLength = clientConfig.QueueLength

	// Log the new configuration (optional, adjust logging as needed)
	s.logger.Infof("Reconfigured MQTT_Client with topic: %s, host: %s, port: %d, qos: %d", s.Topic, s.Host, s.Port, s.QoS)

	// Error handling channel
	errChan := make(chan error, 1)

	// Start InitMQTTClient in a goroutine
	go func() {
		errChan <- s.InitMQTTClient(ctx)
		close(errChan)
	}()

	// Handle errors from the goroutine
	for err := range errChan {
		if err != nil {
			// Handle error, e.g., log it or restart the initialization process
			s.logger.Errorf("Error initializing MQTT client: %v", err)
			// Take appropriate action based on the error
		}
	}

	return err
}

// Get sensor reading
func (s *MQTT_Client) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// Return the latest message if the message queue is not empty and call is from viam data manager
	if extra[data.FromDMString] == true {
		if len(s.messageQueue) != 0 {
			oldestMessage := s.messageQueue[0]
			s.messageQueue = s.messageQueue[1:]
			return map[string]interface{}{
				"payload":            oldestMessage,
				"qos":                int32(s.QoS),
				"topic_from_message": s.Topic,
			}, nil
		} else {
			return nil, data.ErrNoCaptureToStore
		}
	}

	return map[string]interface{}{
		"payload":            s.latestMessage,
		"qos":                int32(s.QoS),
		"topic_from_message": s.Topic,
	}, nil
}

// DoCommand can be implemented to extend sensor functionality but returns unimplemented in this example.
func (s *MQTT_Client) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, errUnimplemented
}

// New function to initialize MQTT client and start the goroutine
func (s *MQTT_Client) InitMQTTClient(ctx context.Context) error {
	// Create a client and connect to the broker
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", s.Host, s.Port))
	// opts.SetClientID("your_client_id") // Set a unique client ID

	s.client = mqtt.NewClient(opts)
	if token := s.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	// Start the goroutine to listen to the topic
	go func() {
		if token := s.client.Subscribe(s.Topic, s.QoS, func(client mqtt.Client, msg mqtt.Message) {
			s.mutex.Lock()
			defer s.mutex.Unlock()

			// Parse the payload
			var payloadData map[string]interface{} // Use this line if the structure is dynamic
			// var payloadData SensorData // Use this line if the structure is known
			err := json.Unmarshal(msg.Payload(), &payloadData)
			if err != nil {
				s.logger.Errorf("error parsing JSON payload:", err)
				return
			}
			s.latestMessage = payloadData
			s.logger.Infof("message queue length: %v", len(s.messageQueue))
			if len(s.messageQueue) == s.queueLength {
				s.messageQueue = s.messageQueue[1:]
				s.messageQueue = append(s.messageQueue, payloadData)
			}
			s.messageQueue = append(s.messageQueue, payloadData)

		}); token.Wait() && token.Error() != nil {
			// Handle subscription error
			s.logger.Errorf("subscription error:", token.Error())
		}
	}()

	return nil
}

// Add a Close method to clean up the MQTT client
func (s *MQTT_Client) Close(ctx context.Context) error {
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(250) // Timeout in milliseconds
	}
	return nil
}
