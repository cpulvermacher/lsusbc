#!/usr/bin/env bash

wget http://www.linux-usb.org/usb.ids

grep '^\w.*' usb.ids >svid.ids
