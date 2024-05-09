# Update

The Simple IoT update client facilitates updating software. Currently, it is
designed to download images for use by the
[Yoe Updater](https://github.com/YoeDistro/yoe-distro/blob/master/docs/updater.md).

<img src="assets/update.png" alt="updater ui" style="zoom:50%;" />

There are several options:

- **Update server**: HTTP server that contains the following files:
  - files.txt: contains a list of update files on the server
  - update files named: `<prefix>_<version>.upd`
    - _version_ should follow [Semantic Versioning](https://semver.org/):
      `MAJOR.MINOR.PATCH`
    - _prefix_ must match what the updater on the target device is expecting --
      typically host/machine name.
- **Prefix**: described above -- typically host/machine name. This is
  autodetected on first startup, but can be changed if necessary.
- **Auto download**: option to periodically check the server for new updates and
  download the latest version.
- **Auto reboot/install**: option to auto install/reboot if a new version is
  detected and downloaded.
