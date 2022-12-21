# yak

[![Build Status](https://github.com/redbubble/yak/actions/workflows/ci.yml/badge.svg)](https://github.com/redbubble/yak/actions)

A tool to generate access keys for AWS using Okta. If you want a backronym, try 'Your AWS Kredentials'.

## Usage

### Installation

We produce builds of `yak` for macOS and Linux.

#### macOS with Homebrew

The easiest option for macOS users is to install `yak` via Homebrew.
This will also help keep `yak` up-to-date when you run `brew upgrade`
as usual.

```sh
brew tap redbubble/redbubble
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
curl -Lq https://raw.githubusercontent.com/redbubble/yak/master/static/delivery-engineers.pub.asc | sudo gpg --no-default-keyring --import --keyring gnupg-ring:/etc/apt/trusted.gpg.d/redbubble.gpg
sudo chmod a+r /etc/apt/trusted.gpg.d/redbubble.gpg
echo "deb http://apt.redbubble.com/ stable main" | sudo tee /etc/apt/sources.list.d/redbubble.list
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

Download the [latest release](https://github.com/redbubble/yak/releases/latest) for your architecture. The `yak` executable is statically linked,
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

Note that to pass `-/--` flags to commands you want to run, you'll need to put a `--` before the
`<command>`, to let `yak` know you're done passing flags to *it*, like this:

```
yak [flags] <role> -- <command --with-flags>
```

For example:

```
yak --cache-only nonprod -- npx cdk --app 'npx ts-node --prefer-ts-exts bin/my-stack.ts' list
```


#### Arguments

```
  -d, --aws-session-duration int        The session duration to request from AWS (in seconds)
      --cache-only                      Only use cache, do not make external requests. Mutually exclusive with --no-cache
      --clear-cache                     Delete all data from yak's cache. If no other arguments are given, exit without error
  -h, --help                            Display this help message and exit
  -l, --list-roles                      List available AWS roles and exit
      --no-cache                        Ignore cache for this request. Mutually exclusive with --cache-only
      --okta-aws-saml-endpoint string   The app embed path for the AWS app within Okta
      --okta-domain string              The domain to use for requests to Okta
      --okta-mfa-provider string        The Okta MFA provider name for login
      --okta-mfa-type string            The Okta MFA type for login
  -u, --okta-username string            Your Okta username
  -o, --output-format string            Can be set to either 'json' or 'env'. The format in which to output credential data
      --pinentry                        Use the pinentry to prompt for credentials, instead of terminal (useful for GUI applications)
      --version                         Print the current version and exit
      --                                Terminator for -/-- flags. Necessary if you want to pass -/-- flags to commands
```

#### Environment Variables

| Variable        | Effect                                                                                     |
|-----------------|--------------------------------------------------------------------------------------------|
| `OKTA_PASSWORD` | The value set in this variable will be passed to Okta as the 'password' component of login |

Please note that setting the `OKTA_PASSWORD` variable in plain text, especially on the command-line, is not a good idea
from a security perspective. A suggested mode of use for this variable would be something like:

```
OKTA_PASSWORD=$(get-password-from-password-manager) yak ...
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
# Yak supports the following values for mfa_type: token:software:totp, token:hardware or push
# For a full list of Okta-supported factors and providers see [this page](https://developer.okta.com/docs/api/resources/factors#supported-factors-for-providers)
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

#### Other Config

```toml
[login]
# Optional. Duration in seconds from the start of the login process until it times out.
timeout = 180
```

```toml
# Optional. Prompt for password and MFA token using pinentry.  Useful for when using GUI tools like Lens.
pinentry = true
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

To hack on `yak`, you'll want to get a copy of the source.  Then:

```
go build
```

### Releasing changes

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

Just run:

```
make test
```

If `gotestsum` isn't available we'll try and install it.
To run tests without gotestsum, or to run the tests for any individual package, you can run:

```
go test <package-directory>
```

## License

`yak` is provided under an MIT license. See the [LICENSE](https://github.com/redbubble/yak/blob/master/LICENSE) file for
details.
