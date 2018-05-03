# yak

A tool to log in to AWS through Okta. If you want a backronym, try 'Your AWS Kredentials'.

## Usage

To install yak, run `go get -u github.com/redbubble/yak`.

### Running

`yak` can be run like this:

```
yak <role> [<command>]
```

and will run `command` as `role`.

More specifically, `yak` runs `command` in the same environment it was called from, with the credentials for `role`
injected as environment variables.

When run without a command, `yak` prints those variables as `export` statements; this is intended to allow easy sourcing
into your shell.

If run with the `--list-roles` flag like this:

```
yak --list-roles
```

`yak` will print a list of available roles and exit.

#### Flags

```
  -d, --aws-session-duration int   The session duration to request from AWS (in seconds)
  -h, --help                       Display this help message and exit
  -l, --list-roles                 List all available AWS roles and exit
      --no-cache                   Do not use caching for this request
  -u, --okta-username string       Your Okta username
```

### Configuring

Yak can be configured with a configuration file at  `~/.yak/config.toml`.

#### Okta Config

```toml
[okta]
# Required. The URL for your okta domain.
domain = "https://<my_okta_domain>.okta.com"

# Required. The path for fetching the SAML assertion from okta.
aws_saml_endpoint = "/app/amazon_aws/<saml_app_id>/sso/saml"

# Optional. Your okta username.
username = "<my_okta_username>"
```

##### How to find your config values

`domain`: This the same domain where you log in to Okta.

`aws_saml_endpoint`: To get this value, you'll need to:

1. Log in to Okta
2. Find the AWS application
3. Copy the URL for the AWS application, e.g. by right-clicking and selecting
   "Copy Link Address" or similar
4. Remove everything up to `okta.com/` (inclusive)
5. Remove everything from the `?` onwards

OR ask your organisation's Okta administrator.

If you're an Okta administrator, you can also:

1. Log in to Okta
2. Click the "Admin" button
3. Navigate to Applications
4. Open the "Amazon Web Services" application
5. On the General tab, copy the App Embed Link
6. Remove everything up to `okta.com/` (inclusive)

`username`: The username you use when logging in to Okta. If in doubt, consult
your organisation's Okta administrator.

#### AWS Config

```toml
[aws]
# Optional. Duration in seconds for the AWS credentials to last. Default 1 hour, maximum 12 hours.
session_duration = 3600
```

#### Aliases

You can configure *role aliases* in the `[alias]` section of your config file; these can be used instead of having to
remember the whole ARN:

```toml
[alias]
prod = "arn:aws:some:long:role:path"
```

This configuration would allow you to log in with:
```
yak prod [<command>]
```

## Development

### Installing dependencies

You'll need [dep](https://github.com/golang/dep) (If you're on OSX, `brew install dep`. Linux is a bit tricker; see the
[README](https://github.com/golang/dep#installation) for details)

Then run:
```
make vendor
```

This will install all your dependencies into the `vendor` directory.

### Running locally

The `make install` target will compile the application and 'install' it into your `$GOPATH`.

You can then run `$GOPATH/bin/yak`.

### Running tests

To run all the tests in the project through [go-passe](https://github.com/redbubble/go-passe), run:
```
make test
```

To run them without go-passe, or to run the tests for any individual package, you can run:
```
go test <package-directory>
```

## License

`yak` is provided under an MIT license. See the [LICENSE](https://github.com/redbubble/yak/blob/master/LICENSE) file for
details.
