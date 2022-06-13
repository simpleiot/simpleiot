# Frequently Asked Questions

### Q: How is this project different than Home Assistant, OpenHAB, Domoticz, etc.?

Although there may be some overlap and Simple IoT may eventually support a
number of off the shelf consumer IoT devices, then genesis of the project and
intent is really for developing IoT products and the infrastructure required to
support them.

### Q: How is this project different than Particle.io, etc.?

Particle.io provides excellent infrastructure to support their devices and solve
many of the hard problems such as remote FW update, getting data securely from
device to cloud, efficient data bandwidth usage, etc. But they don't provide a
way to provide a user facing portal for a product that customers can use to see
data and interact with the device.

### Q: How is this project different than AWS/Azure/GCP/... IoT?

SIOT is designed to be simple to develop and deploy without a lot of moving
parts. We've reduced an IoT system to a
[few basic concepts](https://github.com/simpleiot/simpleiot/tree/master#core-ideas)
that are exactly the same in the cloud and on edge devices. This symmetry is
powerful and allows us to easily implement and move functionality wherever it is
needed. If you need
[Google Scale](https://blog.bradfieldcs.com/you-are-not-google-84912cf44afb),
SIOT may not be the right choice; however, for smaller systems where you want a
system that is easier to develop, deploy, and maintain, consider SIOT.
