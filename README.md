# yak

A tool to log in to AWS through Okta. If you want a backronym, try 'Your AWS Kredentials'.

## Usage

To install yak run `go get -u https://github.com/redbubble/yak`.

TODO non-go-get method of install.

### Configuring

Yak can be configured with a configuration file at  `~/.yak/config.toml`.

```toml
[okta]
# Required. The URL for your okta domain.
domain = "https://<my_okta_domain>.okta.com"

# Required. The path for fetching the SAML assertion from okta.
aws_saml_endpoint = "/app/amazon_aws/<saml_app_id>/sso/saml"

# Optional. Your okta username.
username = "<my_okta_username>"

[aws]
# Optional. Duration in seconds for the AWS credentials to last. Default 1 hour, maximum 12 hours.
session_duration = 3600
```

### Running

TODO

## Development

### Installing dependencies

You'll need [dep](https://github.com/golang/dep) (basically `brew install dep` or `go get github.com/golang/deb`).

Then run:
```
make vendor
```

This will install all your dependencies into the `vendor` directory.

### Running locally

The `make install` target will compile the application and 'install' it into your `$GOPATH`.

You can then run `$GOPATH/bin/yak`.

### Running tests

All the tests in the project can be run with:
```
make test
```

If you'd like to run the tests for a single package, you can run:
```
go test <package-directory>
```

## License

`yak` is provided under an MIT license. See the [LICENSE](https://github.com/redbubble/yak/blob/master/LICENSE) file for
details.
