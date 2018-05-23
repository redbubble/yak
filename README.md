# yak

[![Build Status](https://travis-ci.org/redbubble/yak.svg?branch=master)](https://travis-ci.org/redbubble/yak)

A tool to generate access keys for AWS using Okta. If you want a backronym, try 'Your AWS Kredentials'.

## Usage

### Installation

We produce builds of `yak` for OSX and Linux. Windows is not currently supported.

#### macOS with Homebrew

The easiest option for macOS users is to install `yak` via Homebrew.
This will also help keep `yak` up-to-date when you run `brew upgrade`
as usual.

```sh
brew tap redbubble/yak
brew install yak
```

This will also put ZSH and Bash completions in the right spot; they
should be usable next time you reload your shell config.

#### Ubuntu/Debian APT repository

`yak` can be installed from our APT repo.  This should get you up and
running:

```sh
sudo apt install curl gnupg2
# This is the Redbubble GPG key, to verify releases:
curl https://raw.githubusercontent.com/redbubble/yak/master/static/delivery-engineers.pub.asc | sudo apt-key add -
echo "deb http://apt.redbubble.com/ stable main" | sudo tee /etc/apt/sources.list.d/yak.list
sudo apt update
sudo apt install yak
```

#### Standalone DEB/RPM packages

We generate Deb and RPM packages as part of our release.

Download the package appropriate for your distro from the [latest
release](https://github.com/redbubble/yak/releases/latest) page.
Unfortunately, this won't give you nice automatic updates.

#### A note about completions

We've seen issues using tab-completion on older versions of ZSH.  It seems
that version 5.1 or newer will work correctly.

#### Manually

Download the [latest release](https://github.com/yak/releases/latest) for your architecture. The `yak` executable is statically linked,
so all you should need to do is put the executable somewhere in your `$PATH`.

This method will not give you tab-completion; if you'd like that, the completions files are available in
[/static/completions](https://github.com/redbubble/yak/tree/master/static/completions).

### Running

You can run `yak` like this:

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

Note that if you want to pass -/-- flags to subcommands, you'll need to put a '--' before the <role> to let `yak` know
you're done passing flags to *it*, like this:

```
yak [flags] -- <role> <command --with-flags>
```

#### Arguments

```
  -d, --aws-session-duration int        The session duration to request from AWS (in seconds)
      --cache-only                      Only use cache, do not make external requests. Mutually exclusive with --no-cache
  -h, --help                            Display this help message and exit
  -l, --list-roles                      List available AWS roles and exit
      --no-cache                        Ignore cache for this request. Mutually exclusive with --cache-only
      --okta-aws-saml-endpoint string   The app embed path for the AWS app within Okta
      --okta-domain string              The domain to use for requests to Okta
  -u, --okta-username string            Your Okta username
      --okta-mfa-type string            The Okta MFA type for login
      --okta-mfa-provider string        The Okta MFA provider name for login
      --version                         Print the current version and exit
      --                                Terminator for -/-- flags. Necessary if you want to pass -/-- flags to subcommands
```

### Configuring

Yak can be configured with a configuration file at  `~/.config/yak/config.toml` (`~/.yak/config.toml` is also supported).

#### Okta Config

```toml
[okta]
# Required. The URL for your okta domain.
domain = "https://<my_okta_domain>.okta.com"

# Required. The path for fetching the SAML assertion from okta.
aws_saml_endpoint = "/home/<okta_app_name>/<generic_id>/<app_id>"

# Optional. Your okta username.
username = "<my_okta_username>"

# Optional. Your okta MFA device type and provider so that you don't have to choose.
mfa_type = "<mfa_type>"
mfa_provider = "<mfa_provider>"
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

### Go Get

To hack on `yak`, you'll want to get a copy of the source.  To do
that:

```
go get -u github.com/redbubble/yak
 ```

### Installing dependencies

You'll need the [dep](https://github.com/golang/dep) tool (if you're
on macOS, `brew install dep`. Linux is a bit tricker; see the [dep
README](https://github.com/golang/dep#installation) for details).

Then run (inside your `$GOPATH/src/github.com/redbubble/yak` directory):
```
make vendor
```

This will install all your dependencies into the `vendor` directory.

If you want to do releases, you'll also want the `deb-s3` package.
You'll also want `gnupg2` to be able to sign releases, but i'll leave
installation of that up to you.

```sh
gem install deb-s3
```

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
