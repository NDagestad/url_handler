url_handler(1)

# NAME

url_handler - opens your urls with whatever program you have configured for it

# SYNOPSIS

*url_handler* [OPTIONS] URIs...

_command_ | *url_handler* [OPTIONS] (read URIs from stdin)

*url_handler* [OPTION] (will read from the clipboard)

# DESCRIPTION

*url_handler* is a utility that takes one or more URI and open the coresponding resource
with the program you have configured it to use, in the ways I would have wanted
*xdg-open* to work.
Taking some inspiration from plan 9's plumber, the data given to the
*url_handler* does not strictly have to a URI, with the right regex it should be
possible to use strings like *page(X)* to open the man page of page in section
X or more or less whatever you can imagine.

# OPTIONS

*-loglevel* {*none*|*warning*|*error*|*debug*}
	Log verbosity. Default to _none_

*-help*
	Print the usage and exit.

*-version*
	Print the version and exit.

# FILES

*url_handler* follows the XDG_BASE_DIRECTORY specification so configuration
files should be in *$XDG_CONFIG_HOME*, see *url_handler*(5) for more on that.

# SEE ALSO

*url_handler*(5)

# AUTHORS

Maintained and developped by Nicolai Dagestad <nicolai@dagestad.fr>. The
upstream sources are at https://sr.ht/~nicolai/url_handler. Bug reports
and patches are welcome and should be sent to the mailing list at 
~nicolai_dagestad/url_handler@lists.sr.ht
