# Documentation

Good documentation is critical for any project and to get good documentation,
the process to create it must be as frictionless as possible. With this in mind,
we've structured SIOT documentation as follows:

- Markdown is the primary source format.
- Documentation lives in the same repo as the source code. When you update the
  code, update the documentation at the same time.
- Documentation is easily viewable in GitHub, or our
  [generated docs site](https://docs.simpleiot.org/). This allows any snapshot
  of SIOT to contain a viewable snapshot of the documentation for that revision.
- `mdbook` is used to generate the documentation site.
- All diagrams are stored in a
  [single draw.io](https://github.com/simpleiot/simpleiot/blob/master/docs/diagrams.drawio)
  file. This allows you to easily see what diagrams are available and easily
  copy pieces from existing diagrams to make new ones. Then generate a PNG for
  the diagram in the `images/` directory in the relevant documentation
  directory.
