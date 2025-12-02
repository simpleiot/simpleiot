# Rules

Rules are defined by nodes and are composed of additional child nodes for
conditions and actions. See the node/point
[schema](https://github.com/simpleiot/simpleiot/blob/master/client/rule.go) for
more details.

All points should be sent out periodically, even if values are not changing to
indicate a node is still alive and eliminate the need to periodically run rules.
Even things like system state should be sent out to trigger device/node offline
notifications.
