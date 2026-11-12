# Changelog

All notable changes to the `simpleiot` Python package are documented here. The
format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

- **Set up nodes from a YAML file with `provision()`.** It uploads the file for
  the server to apply, as `siot import` does, and waits until it has, raising
  the server's error if it fails.
- **Autonomous rover example.** A simulated rover reports its position, speed,
  and battery to a dashboard with a live map and takes commands from it. It sets
  up its own nodes with `provision()`. See `examples/rover`.

## [0.1.0]

- **First release.** Connect with the shared token, read and write nodes and
  points, create, move, and delete nodes, and subscribe to point changes.
