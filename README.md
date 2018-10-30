# Simple IoT

Simple IoT is collection of best practices for building IoT applications
learned from experience building real-world applications.

## Vision

- built around collecting and visualizing data
- granular user/organization support
- simple deployment (not a lot of pieces on the backend to manage)
- rules engine that is easy to configure (conditions/consequences)
- plugin architecture for extending system

## Short term features

- initially use http(s) transport for everything
- App Engine Deployment

## Long term features

- efficient protocols for cellular data connections (CoAP, etc)

## Technology choices

Choices for the technology stack emphasizes simplicity, not only in the
language, but just as important in the deployment and tooling.

- Go for backend
  - simple language and deployment model
  - very productive language
  - nice balance of safety + productivity
- Elm for frontend
  - nice balance of simplicity and safety
  - reduces possibility for run time exceptions in browser
  - does not require a huge/fragile build system typical in
    Javascript frontends.
- Google App Engine
  - initial deployment target is simple to deploy applications to,
    and they handle high-availability, scaling, etc.
