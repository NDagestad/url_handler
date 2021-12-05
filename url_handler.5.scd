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
	if it is an executable. Defaults to unset.

*detach_on_exec*
	Not used yet, will make *url_handler* not wait for the end of whatever
	process it launches. Defaults to unset.


All the options specifying an executable can handle arguments. 

# NAMED SECTION

Each named section corresponds to a type of ressource to handle.
The name of the section has not really any importance, but it is the name of the
default filter to execute if none is specified with the *filter* option.
Here as well, arguments specifying executables can handle argments.
All options except for *exec* can appeare multiple times. They can also contain
a coma (,) separated list of values. for *url_regex*s this is not recommended
though as this might not work properly with the regex. (Has not been tested,
there might be a bug there) All these values defaults to unset.

The matching process follows the order in which the options are listed here.
On the first match, the search is stopped and the *exec* corresponding to that 
section is used to handle the URI. The matching process also tries each
ressource type in the order they are listed in the config file.

*exec*
	The only mandatory option, and the only one that must be unique for the
	section. it is the program used to handle the URI.

*protocol*
	The protocols for this ressource.

*extensions*
	The extensions that correspond to this ressource.

*mime_type*
	The mime types asociated with this ressource. (Not yet implemented)

*url_regex*
	Regexs that matche URIs for this ressource.

*filter*
	Program to run on the URI. This programm must return 0 if the URI should
	match for this ressource type and 1 if it should not match. Any other value
	is treated a if an error occured.

	A context is passed to the filter in the following environment variables:
	- *url* the raw URI beeing handled
	- *protocol* the protocol of this URI
	- *user* the user of this URI
	- *host* the host part of this URI
	- *path* the path of the ressource in this URI
	- *section* the name of the section that called the filter

	And lastly, if there is a password in the URI, it is send to the filter over
	it's stdin.


# FILES

*url_handler* (tries to) follow the XDG_BASE_DIRECTORY specification. THe
configuration file should therefor be in $XDG_CONFIG_HOME. 
When looking for it's config file url_handler will look for these files in
this order:
	- *$XDG_CONFIG_HOME/url_handler/config.ini*
	- *$XDG_CONFIG_DIRS/url_handler/config.ini*

(If they are unset, *$XDG_CONFIG_HOME* will normaly  default to *$HOME/.config* and
*$XDG_CONFIG_DIRS* to */etc/xdg*)



# SEE ALSO

*url_handler*(1)
