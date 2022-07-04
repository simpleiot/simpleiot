# Application Architecture

The Simple IoT Go application is a single binary with embedded assets. The
database and NATS server are also embedded by default for easy deployment. There
are five main parts to a Simple IoT application:

1. **NATS Message Bus**: all data goes through this making it very easy to
   observe the system.
2. **Store**: persists the data for the system, merges incoming data, maintains
   node hash values for synchronization, rules engine, etc. (the rules engine
   may eventually move to a client)
3. **Clients**: interact with other devices/systems such as Modbus, 1-wire, etc.
   This is where most of the functionality in a SIOT system lives, and where you
   add your custom functionality. Clients can exist inside the Simple IoT
   application or as external processes written in any language that connect via
   NATS. Clients are represented by a node (and optionally child nodes) in the
   SIOT store. When a node is updated, its respective clients are updated with
   the new information. Likewise, when a client has new information, it sends
   that out to be stored and used by other nodes/instances as needed.
4. **HTTP API**: provides a way for HTTP clients to interact with the system.
5. **Web UI**: Provides a user interface for users to interact with the system.
   Currently it uses the HTTP API, but will eventually connect directly to NATS.

The simplicity of this architecture makes it easy to extend with new
functionality by writing a new client. Following the constraints of
[storing data](data.md) as nodes and points ensures all data is visible and
readable by other clients, as well as being automatically synchronized to
upstream instances.

![application architecture](images/arch-app.png)

## Application Lifecycle

Simple IoT uses the
[`Start()/Stop()`](https://community.tmpdir.org/t/structuring-go-apps-for-testing-and-lifecycle-management-the-start-stop-pattern/550)
pattern for any long running processes. With any long running process, it is
important to not only Start it, but also to be able to cleanly Stop it. This is
important for testing, but is also good practice. Nothing runs forever so we
should never operate under this illusion. The
[oklog/run](https://github.com/oklog/run) packaged is used to start and shutdown
these processes concurrently. Dependencies between processes should be minimized
where possible through retries. If there are hard dependencies, these can be
managed with `WaitStart()/WaitStop()` functions. See
[`server.go`](https://github.com/simpleiot/simpleiot/blob/master/server/server.go)
for an example.

NATS lends itself very well to a decoupled application architecture because the
NATS clients will buffer messages for some time until the server is available.
Thus we can start all the processes that use a NATS client without waiting for
the server to be available first.

Long term, a NATS API that indicates the status of various parts (rules engine,
etc.) of the system would be beneficial. If there are dependencies between
processes, this can be managed inside the process instead of in the code that
starts/stops the processes.

## User Interface

Currently, the User Interface is implemented using a Single Page Architecture
(SPA) Web Application. This keeps the backend and frontend implementations
mostly independent. See [User Interface](../user/ui.md) and
[Frontend](frontend.md) for more information.

There are many web architectures to chose from and web technology is advancing
at a rapid pace. SPAs are not in vogue right now and more complex architectures
are promoted such as Next.js, SveltKit, Deno Fresh, etc. Concerns with SPAs
include large initial load and stability (if frontend code crashes, everything
quits working). These concerns are valid if using Javascript, but with Elm these
concerns are minimal as Elm compiles to very small bundles, and run time
exceptions are extremely rare. This allows us to use a simple web architecture
with minimal coupling to the backend and minimal build complexity. And it will
be a long time until we write enough Elm code that bundle size matters.

A decoupled SPA UI architecture is also very natural in Simple IoT as IoT
systems are inherently distributed. The frontend is just another client, much
the same as a separate machine learning process, a downstream instance, a
scripting process, etc.
