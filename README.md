# dagon-go
A framework to build services that process messages from RabbitMQ implemented in Go.

[Design Information](https://github.com/lvdiscordian/dagon-go/blob/master/dagon.md)


Version 0.1

Right now it can accept a handler but for each queue it can only process one message per queue at a time. It will stop
consuming until it's done handling a message and resume consuming a message.  This will setup the necessary topology on
RabbitMQ and will consume messages.

It's missing the following:

- Ability to send a response.
    * Complete MessageResponseWriter implementation. (20%)
    * A Publisher thread to manage out going message. (0%)
- Use a worker pool to handle the messages. (10%)
- A set of standard middleware to handle things like errors. (10%)
- Tests (may need to look at using wabbit to mock out amqp library.)