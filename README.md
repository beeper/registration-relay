# registration-relay
A relay that helps the [iMessage bridge] fetch data from registration providers ([Mac], [iOS]).

[iMessage bridge]: https://github.com/beeper/imessage
[Mac]: https://github.com/beeper/mac-registration-provider
[iOS]: https://github.com/beeper/phone-registration-provider

## Usage
Build the binary (clone + `go build`) or use a Docker image from GHCR (`ghcr.io/beeper/registration-relay`),
then just run it with some environment variables for configuration.

* `REGISTRATION_RELAY_SECRET` (required) - 32 byte secret key as base64, used to authenticate providers when reconnecting.
    * A secret can be generated with `openssl rand -base64 32`
* `REGISTRATION_RELAY_LISTEN` (defaults to :8000) - IP and port to listen on.
* `REGISTRATION_RELAY_METRICS_LISTEN` (defaults to :5000) - IP and port to listen on for Prometheus metrics.

A reverse proxy configured to allow websockets should be pointed at the RELAY_LISTEN port to enable TLS.
The bridge and registration providers can then be pointed at the public address of the reverse proxy.
