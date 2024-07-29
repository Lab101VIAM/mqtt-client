# Viam MQTT Client Module

This Viam module wraps the Eclipse Paho MQTT Golang library into a Viam sensor component, allowing you to connect to a MQTT broker and poll messages. The module supports a configurable message cache to handle message bursts without loosing messages as well as basic message payload parsing.

Feel free to open an issue if you run into any problems or open a pull request if you have useful features or bug fixes!

## How does this module work?

This Viam module contains a Viam sensor component which represents the MQTT client. Using the below mentioned settings, you can configure the clien to connect to a MQTT broker and subscribe to topics.
Once the component has connected to a broker, you can then use the [sensor component APIs](https://docs.viam.com/components/sensor/) to read messages from the queue.
There is one important feature hidden behind the API. If you use the Viam data manager to record messages, it will always take the oldest MQTT message from the queue until the queue is empty. This way you can make sure you don't loose messages during a message burst and can configure a reasonable polling freqency. If you request the readings with your own client using any of our sdks, you will always get the latest message and the message will not be removed but rather returned again in a next request unless overridden in the meantime.

## MQTT Client Configuration:
### Parameters:
  * "qos": If the subscribing client defines a lower QoS level than the publishing client, the broker will transmit the message with the lower QoS level.
     - At most once (QoS 0)
     - At least once (QoS 1)
     - Exactly once (QoS 2) 
  * "topic": In MQTT, Topic refers to a UTF-8 string that filters messages for a connected client. A topic consists of one or more levels separated by a forward slash (topic level separator).
     - myhome/groundfloor/livingroom/temperature: This topic represents the temperature in the living room of a home located on the ground floor.
     - USA/California/San Francisco/Silicon Valley: This topic hierarchy can track or exchange information about events or data related to the Silicon Valley area in San Francisco, California, within the United States.
     - 5ff4a2ce-e485-40f4-826c-b1a5d81be9b6/status: This topic could be used to monitor the status of a specific device or system identified by its unique identifier.
     - Germany/Bavaria/car/2382340923453/latitude: This topic structure could be utilized to share the latitude coordinates of a particular car in the region of Bavaria, Germany.
  * "host": The broker’s hostname/IP
  * "port": The broker’s port
  * "q_length": How many messages are kept before being overwritten
  * "clientid": Optional string to be used to identify the mqtt client
  * "payload": Specify the message payload structure: "string" | "json" // default raw

### Examples:
```json
{
  "qos": 0,
  "topic": "topic",
  "host": "10.1.8.247",
  "port": 1883,
  "q_length": 10,
  "clientid": "" // optional string
  "payload": "string" | "json" // default raw
}

{
  "qos": 0,
  "topic": "aranet/358151004965/sensors/1010EA/json/measurements",
  "host": "10.1.8.247",
  "port": 1883,
  "q_length": 10,
  "payload": "string" | "json" // default raw
}
```
## Publish MQTT Messages

Viam sensor components provide a DoCommand() api for which we have implemented the publish command.

Example command structures:

```json
{"publish":{
    "topic": "topic",
    "qos": 0,
    "retained": false,
    "payload": "hello world"
}}
```

```json
{"publish":{
    "topic": "test",
    "qos": 0,
    "retained": false,
    "payload": "{\"hello\": \"world\"}"
}}
```

