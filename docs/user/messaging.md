# Messaging Services

SIOT can support multiple messaging services.

## Twilio SMS Messaging

Simple IoT supports sending SMS messages using Twilio's
[SMS service](https://www.twilio.com/messaging/sms). Add a **Messaging Service**
node and then configure.

![twilio](images/twilio.png)

## Email Messaging

_will be added soon ..._

## Schema

Below is an export of a Twilio messaging service node:

```yaml
nodes:
  - msgService:
      authToken: your-twilio-auth-token
      description: Twilio SMS
      from: "+12155551212"
      service: twilio
      sid: ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

`service` selects the service; `twilio` is what is supported today and `smtp`
is reserved for email. `sid` and `authToken` are the Twilio account SID and
auth token, and `from` is the number messages are sent from, written as text so
the leading `+` is kept.

Where the node sits in the tree decides which messages it processes, as
described in the [notifications documentation](notifications.md), so a service
that serves a whole company belongs on the company group rather than on any one
device.

An export carries `authToken` as it was entered, so treat a file that contains
messaging service nodes the way you would treat the token itself.
