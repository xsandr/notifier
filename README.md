Notifier
===========

Simple WebSocket notifications through RabbitMQ

## Usage

Run notifier:
```
go run main.go consumer.go registry.go connection.go message.go --addr=:5000
```

Go to [notification page](http://localhost:5000/) and enter your ID

Send notification through RabbitMQ, for example:
```
pip install universalbus
```
and then
```
from universalbus import EventSender
sender = EventSender('guest', 'guest', 'localhost', '/', exchange='notifications')
sender.push_text('user.15', 'notification example', ttl=15*60)  # instead 15, enter your ID
```


## Offline notifications

For offline users, notifications will be moved into personal queue, and delivered after user log in.

For disable offline notifications:
```
go run *.go --ttl=0
```