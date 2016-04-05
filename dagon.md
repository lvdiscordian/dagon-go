#_dagon_
_dagon_ is essentially a scheme to describe how to setup services to consume and share messages using RabbitMQ.

_dagon_ makes use of RabbitMQ's feature that can bind exchanges.  This allows each service to have their own exchange
that it manages; while also having a global exchange that is always available. This global exchange will be a topic
exchange with name: ```main```.

The purpose of this is to act a sort as middle ground between having a single set of
exchanges for all services to attach queues to and from using a single host per service. The former isn't used mainly
for organizational and maintainability reasons.  The latter causes a few difficulties. Mainly, it's makes it difficult
to communicate between services, requiring a new connection for each service.

#_dagon_'s Naming Scheme
A _dagon_ service needs two identifiers, a service name and an instance identifier. These are both strings. The service
name will also be the name of the service's main managed exchange and its type is topic. The instance identifier should
be unique among all other instances each service.  This identifier can then be used to create instance specific channels
of communication.

The queues of a service should also use the service name, but as a prefix seperated by a ```.```
(```servicename.queuename```).

#_dagon_'s Messaging Topology
Each service's managed exchange needs to be bound to the ```main``` exchange like so:
```
|---------------------|                                |-------------------|
|         main        | ------- servicename.# -------> |    servicename    |
|---------------------|                                |-------------------|
```
So, binding the service exchange to the main exchange with the routing key ```servicename.#``` route all messages that
are published to main and have a route with the service name at the beginning will be delivered to the service's
exchange to be delivered to any appropriate queues.  Other than starting with the service name, there are no other
restrictions no the routing keys binding queues to the service exchange.  However if queues want to receive messages not
delivered to it's service exchange should **only** bind to the main exchange and not to any other services exchange.
This is because that a service's exchange is managed by each individual service and my only exist after that service is
running and thus eliminates any chains of dependencies.  _dagon_ also needs to support multiple bindings for each queue.

#_dagon_ Message Format
The body of each message should be valid JSON.  The only thing that should be in the body is relevant information for
the consumer to process.  Any metadata reguarding the message should be placed into the message properties and headers.
Any rpc calls that will expect a response should set the ```reply-to``` and  ```correlation-id``` properties.  Any
response will be published to the main exchange using the value of the ```reply-to``` property as the routing key.  Any
application specific context information should also be sent in the headers of the message.