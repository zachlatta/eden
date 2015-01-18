#!/bin/sh

osascript single-chat.applescript | sed 's/^.//' | sed 's/.$//'
