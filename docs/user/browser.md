# Browser

The browser client enables control and configuration of the
[Yoe Kiosk Browser](https://github.com/YoeDistro/yoe-kiosk-browser) as it is 
when installed as part of Yoe Distro. On changing the configuration, changes 
are saved to `/etc/default/yoe-kiosk-browser` for the browser and 
`/etc/default/eglfs.json` for EGLFS, and the `yoe-kiosk-browser` service is 
restarted automatically.