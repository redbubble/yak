_yak()
{
    local cur prev opts roles cmd_offset nonoption_count
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="-h --help -l --list-roles -u --okta-username --okta-domain --okta-aws-saml-endpoint -d --aws-session-duration --no-cache --cache-only --version"
    roles=$(yak --list-roles --cache-only 2>/dev/null)

    nonoption_count=0

    for (( i=1; i <= COMP_CWORD; i++ )); do
        if [[ ${COMP_WORDS[i]} != -* ]]; then
            (( nonoption_count++ ))
            if [[ $nonoption_count == 2 ]]; then
                cmd_offset=$i
            fi
        fi
    done

    if [[ $nonoption_count < 2 ]]; then
        if [[ ${cur} == -* ]] ; then
            COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
            return 0
        else
            COMPREPLY=( $(compgen -W "${roles}" -- ${cur}) )
            return 0
        fi
    else
        local root_command=${COMP_WORDS[cmd_offset]}
        _command_offset $cmd_offset
        return 0
    fi
}
complete -F _yak yak
