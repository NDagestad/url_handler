url_handler(1)

# NAME

url_handler - opens your urls with whatever program you have configured for it

# SYNOPSIS

*url_handler* [--debug] URIs...

_command_ | *url_handler* [--debug]

*url_handler* [--debug] (will read from the clipboard)

# DESCRIPTION

url_handler is a utility that takes some URI and open the coresponding resource
with the program you have configured it to use.

# FILES

*url_handler* follows the XDG_BASE_DIRECTORY specification so configuration
files should be in XDG_CONFIG_HOME, see *url_handler*(5) for more on that.

# SEE ALSO

*url_handler*(5)

# AUTHORS

Maintained and developped by Nicolai Dagestad <nicolai@dagestad.fr>. The
upstream sources are at https://git.dagestad.fr/~nicolai/url_handler. Bug reports
and patches are welcome and should be sent directly to the author while there is
no mailling list (There should be one at some point)
