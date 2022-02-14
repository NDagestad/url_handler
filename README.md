# url_handler

Give it an url and it will open it how you want.

Browers are anoying so when I can, I open links with dedicated programs. I started out using
url_hanlder.sh (from some package I can't find anymore sorry to you original author), but I didn't
quite like how it worked, so I rewrote it, and the last version of that monstruous bash script looks
like [this](https://git.dagestad.fr/~nicolai/bin/tree/d427aea871ec91b61c73f70d24aeadfad509326a/item/url_handler)

A 200 line bash script isn't that bad, but it is really not the ideal medium to handle something as
complexe as URI handling. I say URI, because this has evolved into a sort of replacement of
xdg-open, by being also able to open paths to local resources, but different 🙃

Upstream is [https://sr.ht/~nicolai_dagestad/url_handler/](https://sr.ht/~nicolai_dagestad/url_handler/) 
in case this is uploaded somwhere else.

# Building

`make`

That should be all you need.

The man pages are compiled with [scdoc](https://git.sr.ht/~sircmpwn/scdoc). It might even be in your
repos ¯\\\_(ツ)\_/¯.
With scodcs installed a simpel `make doc` will get you groff man pages, but if you just want to 
read them, the scdoc format is more or less markdown so it's quit easy to read 👍.

# Installing

I'll get a PKBGUILD going when it is done, for other distros you're on your own (until somebody find
this interesting enough to package it for you.

If there is no installation mehtod for you just put the executable in your path and add a config
file in `$XDG_CONFIG_HOME/url_handler` on your machine. (`XDG_CONFIG_HOME` will most likely be `~/.config`
if you haven't done anything funny to you system, but if you had, you would probably know about the
xdg base directory spec :p)

# Hacking

Patches are welcome, the mailing list is at [~nicolai_dagestad/url_handler@lists.sr.ht](mailto:~nicolai_dagestad/url_handler@lists.sr.ht).
The project page is [https://sr.ht/~nicolai_dagestad/url_handler/](https://sr.ht/~nicolai_dagestad/url_handler/)

