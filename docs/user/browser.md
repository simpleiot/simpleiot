# Browser

The browser client enables control and configuration of the
[Yoe Kiosk Browser](https://github.com/YoeDistro/yoe-kiosk-browser) as it is
when installed as part of Yoe Distro. On changing the configuration, changes are
saved to `/etc/default/yoe-kiosk-browser` for the browser and
`/etc/default/eglfs.json` for EGLFS, and the `yoe-kiosk-browser` service is
restarted automatically.

## Schema

Below is an export of a browser node:

```yaml
nodes:
  - browser:
      debugport: "9222"
      defaultdialogs: 0
      description: Kiosk
      dialogcolor: "#1c1c1c"
      disabled: 0
      disablesandbox: 1
      displaycard: /dev/dri/card0
      exceptionurl: http://localhost:8118/offline.html
      fullscreen: 1
      ignorecerterr: 0
      keyboardscale: 1
      retryinterval: 10
      rotate: 0
      screenresolution: 1920x1080
      touchquirk: 0
      url: http://localhost:8118
```

The point types are lower case throughout, which matches the settings written to
`/etc/default/yoe-kiosk-browser`. Checkboxes are stored as `1` and `0`.
`debugport` is text, so it is quoted; `rotate` and `retryinterval` are numbers.
`displaycard` and `screenresolution` are the two settings that land in
`/etc/default/eglfs.json`.
