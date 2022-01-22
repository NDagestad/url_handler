url_handler(5)

# NAME

*url_handler* - opens your urls with whatever program you have configured for it

# DESCRIPTION

*url_handler* uses the ini format for its configuration. The default section
contains general setting for the programm, each named section is used to group
together how one ressource type should be handled.

# DEFAULT SECTION

This section does not need to be named, in it you will find these options: 

*browser* 
	The executable to use when no other method has been found to open the link.
	Defaults to _xdg-open_

*program_launcher*
	This is the program that is used to launch the others. It is only usefull if
	you don't want *url_handler* to hang while it waits for it's subprocess to exit
	but you can use it for whatewer you want. If it is not set, it will just not 
	be used. Defaults to unset. 

*filter_path*
	The (absolute for now) path in which filters will be loocked for. If unset,
	the filters will be looked up in $PATH. Default to unset.

*filter_shell*
	The shell used to launch the filters. If unset, the filter will be launched as
	if it is an executable (so don't forget your shebangs!). Defaults to unset.

*clipboard_cmd*
	The shell command to use to get the content of the clipboard. Each line will
	be treated as a URI. Defaults to unset.

*detach_on_exec*
	Makes *url_handler* not wait for the end of whatever process it launches.
	Defaults to false.


All the options specifying an executable can handle arguments. 

# NAMED SECTION

Each named section corresponds to a type of ressource to handle.
The name of the section has not really any importance, but it is the name of the
default filter to execute if none is specified with the *filter* option (See
section *FILTERS* for more info on filters).
Here as well, arguments specifying executables can handle argments.
All options except for *exec* can appeare multiple times. They can also contain
a comma (,) separated list of values. For *url_regex*s this is not recommended
though as this might not work properly with the regex. (Has not been tested,
there might be a bug there) All these values defaults to unset.

The matching process follows the order in which the options are listed here.
On the first match, the search is stopped and the *exec* corresponding to that 
section is used to handle the URI. Section are matched against each URI in the
order in which they appear in the configuration file..

*exec*
	The only mandatory option, and the only one that must be unique for the
	section. it is the program used to handle the URI.

*protocol*
	The protocols for this ressource.

*extensions*
	The extensions that correspond to this ressource.

*mime_type*
	The mime types asociated with this ressource. (Only implemented for local
	files for now)

*url_regex*
	Regexs that matche URIs for this ressource.

*filter*
	Program to run on the URI. See section *FILTERS* for more information on
	them.

*match_expression*
	A logical expression controling what parameters are used and how when
	trying to match an URI, see the *MATCHING* section for more detatils.


# FILTERS

Matching URIs is a complex task, and sometimes the matching behaviour
*url_handler* proposes is not flexible enough. For those cases, an external
program can be run to determine if a URI should matche or not for a given
section.

This programm must return 0 if the URI should match for this ressource type 
and 1 if it should not match. Any other value is treated a if an error occured.

The context is passed to the filter in the following environment variables:
- *url* the raw URI beeing handled
- *protocol* the protocol of this URI
- *user* the user of this URI
- *host* the host part of this URI
- *path* the path of the ressource in this URI
- *section* the name of the section that called the filter

And lastly, if there is a password in the URI, it is sent to the filter over
it's stdin.

# MATCHING

By default, a URI is tried matching with the following logic:
```
extension || filter || mime_type || protocol || url_regex
```
So if any of this parameters match, the URI is seen as matching and the program
defined by *exec* is ran.

For more flexibility, this behaviour can be changed by setting another
expression.

A matching expression support a rather classical logique syntax with
parenthesis, *||* for ORing, *&&* for ANDing, and *!* for inverting the value.

You can also use ternary operations with the same syntax as in C:
```
expr ? true : false
```
but you might be going a litle far there, what kind or
twited logic do you need to handle if you have to use conditional matching in
your configuration. 

(More than what is listed here is handle because of the underlying go library
being used, but this is what is most applicable to this utility. If you want to
have fun you can look here https://github.com/PaesslerAG/gval. This program 
uses the defaul language)

The names used in the expression are the same as used for the parameters, except
for *extensions* which is called *extension* in the match expression.


# VARIABLES

You can defines variables in the *DEFAULT SECTION* that can then be reused in
values in other section like so:

```
name=VALUE
.
.
.
[filter]
exec=executable %(name)s args
mime_type=file/whatever
```

# FILES

*url_handler* (tries to) follow the XDG_BASE_DIRECTORY specification. THe
configuration file should therefor be in *$XDG_CONFIG_HOME*. 
When looking for it's config file url_handler will look for these files in
this order:
	- *$XDG_CONFIG_HOME/url_handler/config.ini*
	- *$XDG_CONFIG_DIRS/url_handler/config.ini* 

*$XDG_CONFIG_DIRS* is a list, each directory will be tried until a config file is
found.

(If they are unset, *$XDG_CONFIG_HOME* will normaly default to *$HOME/.config* and
*$XDG_CONFIG_DIRS* to */etc/xdg*)

# NOTE

The default values are not necessarily the same values as in the config file.
This could be seen as a bug, but it is to make the program more usable right
after installation.

# BUGS

If a *url_regex* contains a comma (,), the ini parser will probably try to split
it and treat it as 2 separate *url_regex* entries. I have not tested this so it 
remains to be confirmed, maybe the comma can be escaped. If somebody can confirm 
test it and send a patch if the bug is confirmed that would be greatly apreciated 👍

If a *URI* passed to the programm contains non-printable caracters, the program
might crash.

# SEE ALSO

*url_handler*(1)
