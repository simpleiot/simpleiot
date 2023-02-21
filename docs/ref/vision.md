# Vision

This document attempts to outlines the project philosophy and core values. The
basics are covered in the [readme](../../). As the name suggests, a core value
of the project is simplicity. Thus any changes should be made with this in mind.
Although this project has already proven useful on several real-world project,
it is a work in progress and will continue to improve. As we continue to explore
and refine the project, many things are getting simpler and more flexible. This
process takes time and effort.

> “When you first start off trying to solve a problem, the first solutions you
> come up with are very complex, and most people stop there. But if you keep
> going, and live with the problem and peel more layers of the onion off, you
> can often times arrive at some very elegant and simple solutions.” -- Steve
> Jobs

## Guiding principles

1. Simple concepts are flexible and scale well.
1. IoT systems are inheriently distributed, and distrbuted systems are hard.
1. There are more problems to solve than people to solve them, thus it makes
   sense to collaborate on the common technology pieces.
1. There are a lot of IoT applications that are
   [not Google](https://blog.bradfieldcs.com/you-are-not-google-84912cf44afb)
   scale (10-1000 device range).
1. There is significant opportunity in the
   [long tail](https://www.linkedin.com/pulse/long-tail-iot-param-singh) of IoT,
   which is our focus.
1. There is value in custom solutions (programming vs drag-n-drop).
1. There is value in running/owning our [own platform](https://tmpdir.org/014/).
1. A single engineer should be able to build and deploy a custom IoT system.
1. We don't need to spend excessive amounts of time on operations. For smaller
   deployments, we deploy one binary to a cloud server and we are done with
   operations. We don't need 20 microservices when one
   [monolith](https://m.signalvnoise.com/the-majestic-monolith/) will
   [work](https://changelog.com/posts/monoliths-are-the-future) just
   [fine](https://m.signalvnoise.com/integrated-systems-for-integrated-programmers/).
1. For many applications, a couple hours of down time is not the end of the
   world. Thus a single server that can be quickly rebuilt as needed is adequate
   and in many cases more reliable than complex systems with many moving parts.

## Technology choices

Choices for the technology stack emphasize simplicity, not only in the language,
but just as important, in the deployment and tooling.

- **Backend**
  - [Go](https://golang.org/)
    - simple language and deployment model
    - nice balance of safety + productivity
    - excellent tooling and build system
    - see
      [this thread](https://community.tmpdir.org/t/selecting-a-programming-language/98)
      for more discussion/information
- **Frontend**
  - Single Page Application (SPA) architecture
    - fits well with real-time applications where data is changing all the time
    - easier to transition to Progressive Web Apps (PWA)
  - [Elm](https://elm-lang.org/)
    - nice balance of safety + productivity
    - excellent compiler messages
    - reduces possibility for run time exceptions in browser
    - does not require a huge/complicated/fragile build system typical in
      Javascript frontends.
    - excellent choice for SPAs
  - [elm-ui](https://github.com/mdgriffith/elm-ui)
    - What if you never had to write CSS again?
    - a fun, yet powerful way to lay out a user interface and allows you to
      efficiently make changes and get the layout you want.
- **Database**
  - SQLite
    - see [Store](store.md)
  - Eventually support multiple databased backends depending on scaling/admin
    needs
- **Cloud Hosting**
  - Any machine that provides ability run long-lived Go applications
  - Any MAC/Linux/Windows/rPI/Beaglebone/Odroid/etc computer on your local
    network.
  - Cloud VMs: Digital Ocean, Linode, GCP compute engine, AWS ec2, etc. Can
    easily host on a \$5/mo instance.
- **Edge Devices**
  - any device that runs Linux (rPI, Beaglebone-black, industrial SBCs, your
    custom hardware ...)

In our experience, simplicity and good tooling matter. It is easy to add
features to a language, but creating a useful language/tooling that is simple is
hard. Since we are using Elm on the frontend, it might seem appropriate to
select a functional language like Elixir, Scala, Clojure, Haskell, etc. for the
backend. These environments are likely excellent for many projects, but are also
considerably more complex to work in. The programming style (procedural,
functional, etc.) is important, but other factors such as
simplicity/tooling/deployment are also important, especially for small teams who
don't have separate staff for backend/frontend/operations. Learning two simple
languages (Go and Elm) is a small task compared to dealing with huge languages,
fussy build tools, and complex deployment environments.

This is just a snapshot in time -- there will likely be other better technology
choices in the future. The backend and frontend are independent. If either needs
to be swapped out for a better technology in the future, that is possible.
