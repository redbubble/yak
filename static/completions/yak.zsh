#compdef _yak yak

function _yak {
    local roles=($(yak --list-roles --cache-only 2>/dev/null))
    _arguments -S \
               '-h[Display this help message and exit]' \
               '--help[Display this help message and exit]' \
               '-l[List available AWS roles and exit]' \
               '--list-roles[List available AWS roles and exit]' \
               '-u[Your Okta username]:username' \
               '--okta-username[Your Okta username]:username' \
               '--okta-domain[The domain to use for requests to Okta]:domain' \
               '--okta-aws-saml-endpoint[The app embed path for the AWS app within Okta]:path' \
               '-d[The session duration to request from AWS (in seconds)]:duration' \
               '--aws-session-duration[The session duration to request from AWS (in seconds)]:duration' \
               '--no-cache[Ignore cache for this request. Mutually exclusive with --cache-only]' \
               '--cache-only[Only use cache, do not make external requests. Mutually exclusive with --no-cache]' \
               '--version[Print the current version and exit]' \
               '1:environment:(${roles})' '*:::command:_normal'
}
