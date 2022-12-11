## KrakenD Authorization Plugin

This package provides the Authorization `proxy/client` plugin for [KrakenD API Gateway](https://krakend.io/).

#### Plugin configuration

[`kauth.json`](kauth.json) is the config file for this plugin, and it should be in the same folder as the plugin exists.

The fields of the configuration files as follows:

- `path`: the URL to auth server the registering endpoint of backend service

### Building plugin

Compile the plugin with `go build -buildmode=plugin -o yourplugin.so`, and then reference them in the KrakenD
configuration file. For instance:

```
//backend part of endpoints
"backend": [
        {
          "method": "POST",
          "encoding": "no-op",
          "host": [
            "http://localhost:8080"
          ],
          "url_pattern": "/some-backend/endpoint",
          "extra_config": {
            "plugin/http-client": {
                "name": "kauth"
             }
          }
        }
      ]
      //rest of the config
```

### Tests

### Run KrakenD

To use the Auth plugin with KrakenD, check the krakend-plugin.json, which is a blueprint for injecting a client plugin.

### Plugin Builder Docker

docker run -it --platform linux/amd64 --rm -v "$PWD/Sites/krakend-plugin:/app" -w /app
devopsfaith/krakend-plugin-builder:2.1.3 go build -buildmode=plugin -o kauth.so .

### Krakend-ce Docker

docker run --rm --platform linux/amd64 --name krakend-ce -p 8080:8080 -v "$PWD/Sites/krakend/config/:/etc/krakend/" -v "
$PWD/Sites/krakend/plugins/:/opt/krakend/plugins/" devopsfaith/krakend run -dc /etc/krakend/krakend.json
