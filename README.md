# Simple IoT

Simple IoT is collection of best practices for building IoT applications
learned from experience building real-world applications.

Demo is running at: https://portal.simpleiot.org/

There is not much here yet -- mostly just a build/deployment system at
this point.

[Detailed Documentation](docs/README.md)

## Vision

- built around collecting and visualizing data from devices
- provide a good base system to build IoT products that may support a number
  devices, users, rules, etc.
- requires coding to customize. This is not a GUI for building IoT systems,
  but rather a code base software developers can use as a starting point.
- application technology is general, so you are not dependant on any one
  IoT company, or cloud provider
- plugin architecture for extending the system
- easy to host yourself with a simple deployment process
  (not a lot of pieces on the backend to manage)
- For small deployments (< 1000 devices), application can run standalone
  (no external databases or services are required). Installation is as simple
  as starting one executable.
- Storage (config and sensor data) supports multiple db backends (mongodb,
  Google Cloud Datastore, influxdb, etc). This allows the application to scale
  to any number of devices.
- All assets are embedded. Deployment/upgrade is as simple as copying one binary.

## Short term features

- Embedded database using boldhold
- user accounts
- support esp32 devices
- store timeseries data
- graph timeseries data
- rules engine (conditions/consequences)

done:

- device management
- simple dashboard for each device showing collected parameters
- REST api for devices

## Long term features

- efficient protocols for cellular data connections (CoAP, etc.)
- Google Cloud Datastore
- App Engine Deployment
- edge computing features
- organization support

## Technology choices

Choices for the technology stack emphasizes simplicity, not only in the
language, but just as important in the deployment and tooling.

- **Backend**
  - Go
    - simple language and deployment model
    - nice balance of safety + productivity
    - excellent tooling and build system
- **Frontend**
  - Single Page Application (SPA) architecture
    - programming environment is much more powerful than server rendered
      pages (PHP, Rails, etc).
    - easier to transition to Progressive Web Apps (PWA)
  - Elm
    - nice balance of safety + productivity
    - excellent compiler messages
    - reduces possibility for run time exceptions in browser
    - does not require a huge/fragile build system typical in
      Javascript frontends.
  - Bootstrap
    - mature CSS toolkit that handles browser differences and
      responsive design for mobile reasonably well.
    - widespread adoption and well understand by many developers
    - well supporting bindings in Elm
- **Hosting**
  - Any server (Digital Ocean, Linode, etc)
  - Google App Engine
    - is simple to deploy Go applications
    - handle high-availability, scaling, etc.
  - (any server/hosting environment that supports Go apps can be used)

In our experience, simplicity and good tooling matter. It is easy to add features
to a language, but creating a useful language/tooling that is simple is even harder.
Since we are using Elm on the frontend, it may have seem appropriate to select
a functional language like Elixir, Scala, Clojure, etc. for the backend. These
environments are likely excellent for many projects, but also considerably more
complex to work in. The programming style (procedural, functional, etc.) are important,
but other factors such as simplicity/tooling/deployment are also important, especially
for small teams who don't have separate staff for backend/frontend/operations. Learning two
simple languages (Go and Elm) is a small task compared to dealing with complex
languages and deployment environments.

This is just a snapshot in time -- there will likely be other better technology choices in the
future. The backend and frontend are independent. If either needs
to be swapped out for a better technology in the future, that is possible.
