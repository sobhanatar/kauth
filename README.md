## KrakenD Authorization Plugin

This package provides the Authorization `proxy/client` plugin for [KrakenD API Gateway](https://krakend.io/).

#### Plugin configuration

[`auth_client_settings.json`](auth_client_settings.json) is the config file for this plugin, and it should be in the
same folder as the plugin
exists.

The fields of the configuration files as follows:

- `log_level`: the level of debug application will log in file
- `timeout`: the time in `milliseconds` that HTTP handler will wait for the response
- `retry_max`: the maximum number of retries
- `retry_wait_min`: the minimum time the client wait in `milliseconds.`
- `retry_wait_max`: the maximum time the client wait in `milliseconds.`
- `url`: the URL to auth server the registering endpoint of backend service

#### Plugin Logging

This package uses [logrus](https://github.com/sirupsen/logrus) for logging failures to file. The file name follows the
pattern of `auth-client-plugin-{date}.log,` and is in JSON format so that it can be consumed by services
like `logstash.`

### Building plugin

Compile the plugin with `go build -buildmode=plugin -o yourplugin.so`, and then reference them in the KrakenD
configuration file. For instance:

```
//backend part of endpoints
"backend": [
        {
          "method": "POST",
          "encoding": "json",
          "host": [
            "http://localhost:8080"
          ],
          "url_pattern": "/some-backend/endpoint",
          "extra_config": {
            "plugin/http-client": {
                "name": "auth-client"
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
docker run -it --platform linux/amd64 --rm -v "/Users/sobhanatar/Sites/krakend-plugin:/app" -w /app devopsfaith/krakend-plugin-builder:2.1.3 go build -buildmode=plugin -o kauth.so .

### Krakend-ce Docker
docker run --rm --platform linux/amd64 --name krakend-ce -p 8080:8080 -v "/Users/sobhanatar/Sites/krakend/config/:/etc/krakend/" -v "/Users/sobhanatar/Sites/krakend/plugins/:/opt/krakend/plugins/" devopsfaith/krakend run -dc /etc/krakend/krakend.json